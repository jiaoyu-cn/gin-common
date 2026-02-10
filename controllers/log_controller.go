package controllers

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type LogController struct {
	Config *LogConfig
}

type LogConfig struct {
	StorageLogPath string // 存储路径，默认为 "./storage/logs"
	MaxFileSize    int64
}

func NewLogController(config *LogConfig) *LogController {
	if config == nil {
		config = &LogConfig{}
	}
	if config.StorageLogPath == "" {
		config.StorageLogPath = "./storage/logs"
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 100 * 1024 * 1024
	}
	return &LogController{Config: config}
}
func (c *LogController) Act(gc *gin.Context) {
	act := gc.Param("act")
	switch act {
	case "download":
		c.download(gc)
	case "ls":
		c.ls(gc)
	case "ps":
		c.ps(gc)
	case "top":
		c.top(gc)
	case "df":
		c.df(gc)
	case "chmod":
		c.chmod(gc)
	case "clean":
		c.clean(gc)
	default:
		c.index(gc)
	}
}

func (c *LogController) index(gc *gin.Context) {
	files := make(map[string][]string)
	err := filepath.Walk(c.Config.StorageLogPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if path == c.Config.StorageLogPath {
			return nil
		}
		relPath, err := filepath.Rel(c.Config.StorageLogPath, path)
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(path)
			if ext != ".log" && ext != ".sql" {
				return nil
			}
			dir := filepath.Dir(relPath)
			files[dir] = append(files[dir], relPath)
		}
		return nil
	})
	if err != nil {
		c.htmlError(gc, "获取日志文件列表失败: "+err.Error())
		return
	}
	tmplContent := `
<!DOCTYPE html>
<html>
<head>
    <title>查看日志</title>
    <script src="https://www.eol.cn/e_js/index/2022/jquery.min.js" ignoreapd="false"></script>
    <style>
        a { text-decoration: none; }
        body { width: 1024px; }
        .menu a { border: #0b0d0f 1px solid; padding: 1px; }
        .body{ clear: both; overflow: hidden; }
        .left{ float: left; }
        .right { float: right; }
        dt{ background-color: rgba(246, 162, 118, 0.76); margin: 5px; padding: 5px; width: 1008px; overflow: hidden; }
        dl dd{ height: 30px; line-height: 30px; width: 100%; }
        dl dd:nth-child(even){ background-color: #e3f2fd; }
        dl dd:hover{ background-color: #b9d9f6; }
        .name { font-weight: bold; display: inline-block; }
        .option { display: inline-block; }
    </style>
</head>
<body>
<div class="menu">
    <div class="left">
        <a target="_blank" href="{{printf "%s/ps?key=www-root" .BaseURI}}">查看进程</a>
        <a target="_blank" href="{{printf "%s/ls?key=" .BaseURI}}">查看目录</a>
        <a target="_blank" href="{{printf "%s/top?key=" .BaseURI}}">性能分析</a>
        <a target="_blank" href="{{printf "%s/chmod?key=" .BaseURI}}">修改权限</a>
        <a target="_blank" href="{{printf "%s/clean?key=" .BaseURI}}">清理目录</a>
        <a target="_blank" href="{{printf "%s/df" .BaseURI}}">df</a>
    </div>
    <div class="right">
        <a href="javascript:;" onclick="logShow(true)">全部展开</a>
        <a href="javascript:;" onclick="logShow(false)">全部隐藏</a>
    </div>
</div>
<div class="body">
{{range $dir, $items := .Files}}
    <dl>
        <dt onclick="logToggle()">
            <div class="left">目录：<strong>{{$dir}}</strong> </div>
            <div class="right"> {{len $items}} 个 </div>
        </dt>
        {{range $item := $items}}
            <dd>
                <div class="name">{{$item}}</div>
                <div class="option">
                    <a target="_blank" href="{{printf "%s/download?file=%s" $.BaseURI $item}}">查看</a>
                    <a target="_blank" href="{{printf "%s/download?file=%s&download=true" $.BaseURI $item}}">下载</a>
                    <a href="{{printf "%s/download?file=%s&remove=true" $.BaseURI $item}}">删除</a>
                </div>
            </dd>
        {{end}}
    </dl>
{{end}}
</div>
<script>
    function logShow(isShow) {
        if(isShow){ $('dd').show(); }else{ $('dd').hide(); }
    }
    $('dt').on('click', function (){ $(this).closest('dl').find('dd').toggle(); });
    $('dl').each(function (i){
        let tmpWidth = 0;
        $(this).find('dd div.name').each(function (j) {
            tmpWidth = Math.max($(this).outerWidth(true), tmpWidth);
        });
        $(this).find('dd div.name').width((tmpWidth + 20) + 'px');
    });
    logShow(false);
</script>
</body>
</html>`
	tmpl, err := template.New("log").Parse(tmplContent)
	if err != nil {
		c.htmlError(gc, "模板解析失败: "+err.Error())
		return
	}
	gc.Header("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.Execute(gc.Writer, map[string]interface{}{
		"BaseURI": strings.TrimSuffix(gc.FullPath(), "/:act"),
		"Files":   files,
	}); err != nil {
		c.htmlError(gc, "模板渲染失败: "+err.Error())
	}
}
func (c *LogController) download(gc *gin.Context) {
	fileParam := gc.Query("file")
	if fileParam == "" {
		c.htmlError(gc, "文件参数不能为空")
		return
	}
	filePath := filepath.Join(c.Config.StorageLogPath, fileParam)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		c.htmlError(gc, "文件不存在")
		return
	}
	if err != nil {
		c.htmlError(gc, "无法获取文件信息")
		return
	}
	if info.IsDir() {
		c.htmlError(gc, "目录不允许操作")
		return
	}
	if gc.Query("remove") == "true" {
		if err := os.Remove(filePath); err != nil {
			c.htmlError(gc, "文件删除失败")
			return
		}
		gc.Redirect(http.StatusFound, strings.Replace(gc.FullPath(), ":act", "index", -1))
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		c.htmlError(gc, "无法打开文件")
		return
	}
	defer file.Close()
	if gc.Query("download") == "true" {
		filename := filepath.Base(filePath)
		gc.Header("Content-Type", "application/octet-stream")
		gc.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		http.ServeContent(gc.Writer, gc.Request, filename, info.ModTime(), file)
		return
	}
	if info.Size() > c.Config.MaxFileSize {
		c.htmlError(gc, "文件过大，不支持流式查看")
		return
	}
	gc.Header("Content-Type", "text/plain; charset=UTF-8")
	writer := gc.Writer
	buffer := make([]byte, 1024*1024)
	var totalRead int64
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			written, _ := writer.Write(buffer[:n])
			totalRead += int64(written)
			if totalRead%((1024*1024)*10) == 0 {
				writer.Flush()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	writer.Flush()
}
func (c *LogController) ls(gc *gin.Context) {
	key := gc.Query("key")
	dir := filepath.Join("./storage", key)
	cmd := exec.Command("ls", "-alh", dir)
	result := fmt.Sprintf("执行命令: %s<br>", cmd.Args)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err == nil {
		if outputStr == "" {
			result += "没有找到匹配的文件或目录"
		} else {
			result += strings.ReplaceAll(outputStr, "\n", "<br>")
		}
	} else {
		if outputStr == "" {
			outputStr = err.Error()
		}
		result += "执行命令失败: " + outputStr
	}
	c.htmlSuccess(gc, result)
}
func (c *LogController) ps(gc *gin.Context) {
	key := gc.Query("key")
	if key == "" {
		c.htmlError(gc, "请输入要搜索的进程关键字")
		return
	}
	if strings.ContainsAny(key, "|&;$\\'\"<>()[]{}!`") {
		c.htmlError(gc, "关键字包含非法字符")
		return
	}
	cmdLine := fmt.Sprintf("ps aux | grep -v grep | grep -e START -e %s", key)
	cmd := exec.Command("sh", "-c", cmdLine)
	result := fmt.Sprintf("执行命令: %s<br>", cmdLine)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err == nil {
		if outputStr == "" {
			result += "没有获取到信息"
		} else {
			result += strings.ReplaceAll(outputStr, "\n", "<br>")
		}
	} else {
		if outputStr == "" {
			outputStr = err.Error()
		}
		result += "执行命令失败: " + outputStr
	}
	c.htmlSuccess(gc, result)
}
func (c *LogController) top(gc *gin.Context) {
	key := gc.Query("key")
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"-l", "1"}
	case "linux":
		args = []string{"-bn1", "-w", "600"}
	default:
		c.htmlError(gc, "不支持的操作系统")
		return
	}
	if key != "" {
		args = append(args, key)
	}
	cmd := exec.Command("top", args...)
	result := fmt.Sprintf("执行命令: %s<br>", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err == nil {
		if outputStr == "" {
			result += "没有获取到信息"
		} else {
			result += strings.ReplaceAll(outputStr, "\n", "<br>")
		}
	} else {
		if outputStr == "" {
			outputStr = err.Error()
		}
		result += "执行命令失败: " + outputStr
	}
	c.htmlSuccess(gc, result)
}
func (c *LogController) df(gc *gin.Context) {
	cmd := exec.Command("df", "-h")
	result := "执行命令: df -h<br>"
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err == nil {
		if outputStr == "" {
			result += "没有获取到信息"
		} else {
			result += strings.ReplaceAll(outputStr, "\n", "<br>")
		}
	} else {
		if outputStr == "" {
			outputStr = err.Error()
		}
		result += "执行命令失败: " + outputStr
	}
	c.htmlSuccess(gc, result)
}
func (c *LogController) chmod(gc *gin.Context) {
	key := gc.Query("key")
	if key == "" {
		c.htmlError(gc, "请输入要修改的权限目录")
		return
	}
	dir := filepath.Join("./storage", key)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		c.htmlError(gc, "文件/目录不存在")
		return
	}
	cmd := exec.Command("chmod", "-R", "770", dir)
	result := fmt.Sprintf("执行命令: %s<br>", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err == nil {
		result += strings.ReplaceAll(outputStr, "\n", "<br>")
	} else {
		if outputStr == "" {
			outputStr = err.Error()
		}
		result += "执行命令失败: " + outputStr
	}
	c.htmlSuccess(gc, result)
}
func (c *LogController) clean(gc *gin.Context) {
	key := gc.Query("key")
	if key == "" {
		key = "7"
	}
	days, err := strconv.Atoi(key)
	if err != nil || days < 3 {
		c.htmlError(gc, "日志保留天数最小为3天")
		return
	}
	var cleanedFiles []string
	cutoffTime := time.Now().AddDate(0, 0, -days)
	err = filepath.Walk(c.Config.StorageLogPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".log" && ext != ".sql" {
			return nil
		}
		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(path); err != nil {
				return nil
			}
			cleanedFiles = append(cleanedFiles, path)
		}
		return nil
	})
	if len(cleanedFiles) == 0 {
		gc.Redirect(http.StatusFound, strings.Replace(gc.FullPath(), ":act", "index", -1))
		return
	}
	c.htmlSuccess(gc, "删除文件如下:<br>"+strings.Join(cleanedFiles, "<br>"))
}
func (c *LogController) htmlSuccess(gc *gin.Context, msg string) {
	gc.Header("Content-Type", "text/html; charset=utf-8")
	gc.String(http.StatusOK, "<pre>"+msg+"</pre>")
}
func (c *LogController) htmlError(gc *gin.Context, msg string) {
	gc.Header("Content-Type", "text/html; charset=utf-8")
	gc.String(http.StatusInternalServerError, "<pre>"+msg+"</pre>")
}
