package main  
  
import (  
    "bufio"  
    "bytes"  
    "fmt"  
    "io"  
    "log"  
    "os"  
    "path/filepath"  
    "regexp"  
    "runtime"  
)  
  
var workers = runtime.NumCPU()  
  
type Result struct {  
    filename string  
    lino     int  
    line     string  
}  
  
type Job struct {  
    filename string  
    results  chan<- Result  
}  
  
func main() {  
  
    // go语言里大多数并发程序的开始处都有这一行代码, 但这行代码最终将会是多余的,  
    // 因为go语言的运行时系统会变得足够聪明以自动适配它所运行的机器  
    runtime.GOMAXPROCS(runtime.NumCPU())  
  
    // 返回当前处理器的数量  
    fmt.Println(runtime.GOMAXPROCS(0))  
    // 返回当前机器的逻辑处理器或者核心的数量  
    fmt.Println(runtime.NumCPU())  
  
    // Args hold the command-line arguments, starting with the program name  
    if len(os.Args) < 3 || os.Args[1] == "-h" || os.Args[1] == "--help" {  
  
        // Base returns the last element of path. Trailing path separators are removed before extracting the last element. If the path is empty, Base returns ".". If the path consists entirely of separators, Base returns a single separator  
        fmt.Printf("usage: %s <regexp> <files>\n",  
            filepath.Base(os.Args[0]))  
        // Exit causes the current program to exit with the given status code. Conventionally, code zero indicates success, non-zero an error. The program terminates immediately; deferred functions are not run  
        os.Exit(1)  
    }  
  
    // Compile parses a regular expression and returns, if successful, a Regexp object that can be used to match against text  
    if lineRx, err := regexp.Compile(os.Args[1]); err != nil {  
        log.Fatalf("invalid regexp: %s\n", err)  
    } else {  
        grep(lineRx, commandLineFiles(os.Args[2:]))  
    }  
}  
  
func commandLineFiles(files []string) []string {  
  
    // Package runtime contains operations that interact with Go's runtime system, such as functions to control goroutines. It also includes the low-level type information used by the reflect package; see reflect's documentation for the programmable interface to the run-time type system  
    // GOOS is the running program's operating system target: one of darwin, freebsd, linux, and so on  
    if runtime.GOOS == "windows" {  
        args := make([]string, 0, len(files))  
        for _, name := range files {  
  
            // Glob returns the names of all files matching pattern or nil if there is no matching file. The syntax of patterns is the same as in Match. The pattern may describe hierarchical names such as /usr/*/bin/ed (assuming the Separator is '/')  
            if matches, err := filepath.Glob(name); err != nil {  
                args = append(args, name) // Invalid pattern  
            } else if matches != nil { // At least one match  
                args = append(args, matches...)  
            }  
        }  
        return args  
    }  
    return files  
}  
  
func grep(lineRx *regexp.Regexp, filenames []string) {  
  
    // 定义需要的channels切片  
    jobs := make(chan Job, workers)  
    results := make(chan Result, minimum(1000, len(filenames)))  
    done := make(chan struct{}, workers)  
  
    // ---------------------------------------------  
    /*  
     * 下面是go协程并发处理的一个经典框架  
     */  
  
    // 将需要并发处理的任务添加到jobs的channel中  
    go addJobs(jobs, filenames, results) // Executes in its own goroutine  
  
    // 根据cpu的数量启动对应个数的goroutines从jobs争夺任务进行处理  
    for i := 0; i < workers; i++ {  
        go doJobs(done, lineRx, jobs) // Each executes in its own goroutine  
    }  
  
    // 新创建一个接受结果的routine, 等待所有worker routiines的完成结果, 并将结果通知主routine  
    go awaitCompletion(done, results)  
  
    // 在主routine输出结果  
    processResults(results)  
    // ---------------------------------------------  
  
}  
  
func addJobs(jobs chan<- Job, filenames []string, results chan<- Result) {  
    for _, filename := range filenames {  
  
        // 在channel中添加任务  
        jobs <- Job{filename, results}  
    }  
    close(jobs)  
}  
  
func doJobs(done chan<- struct{}, lineRx *regexp.Regexp, jobs <-chan Job) {  
  
    // 在channel中取出任务并计算  
    for job := range jobs {  
  
        /* 
         * 定义类型自己的方法来处理业务逻辑 
         */  
        job.Do(lineRx)  
    }  
  
    // 所有任务完成后的结束标志, 一个空结构体切片  
    done <- struct{}{}  
}  
  
// 方法是作用在自定义类型的值上的一类特殊函数  
func (job Job) Do(lineRx *regexp.Regexp) {  
    file, err := os.Open(job.filename)  
    if err != nil {  
        log.Printf("error: %s\n", err)  
        return  
    }  
    // 延迟释放, 类似C++中的析构函数  
    defer file.Close()  
  
    // NewReader returns a new Reader whose buffer has the default size  
    reader := bufio.NewReader(file)  
    for lino := 1; ; lino++ {  
  
        // ReadBytes reads until the first occurrence of delim in the input, returning a slice containing the data up to and including the delimiter. If ReadBytes encounters an error before finding a delimiter, it returns the data read before the error and the error itself (often io.EOF). ReadBytes returns err != nil if and only if the returned data does not end in delim. For simple uses, a Scanner may be more convenient  
        line, err := reader.ReadBytes('\n')  
  
        // Package bytes implements functions for the manipulation of byte slices. It is analogous to the facilities of the strings package  
        // TrimRight returns a subslice of s by slicing off all trailing UTF-8-encoded Unicode code points that are contained in cutset  
        line = bytes.TrimRight(line, "\n\r")  
  
        // Match reports whether the Regexp matches the byte slice b  
        if lineRx.Match(line) {  
  
            // 若匹配则将文件名, 行号, 匹配的行存在结果集里, 结果集是一个管道类型  
            job.results <- Result{job.filename, lino, string(line)}  
        }  
  
        // 读文件出错  
        if err != nil {  
            if err != io.EOF {  
                log.Printf("error:%d: %s\n", lino, err)  
            }  
            break  
        }  
    }  
}  
  
func awaitCompletion(done <-chan struct{}, results chan Result) {  
    for i := 0; i < workers; i++ {  
        <-done  
    }  
    close(results)  
}  
  
func processResults(results <-chan Result) {  
    for result := range results {  
        fmt.Printf("%s:%d:%s\n", result.filename, result.lino, result.line)  
    }  
}  
  
func minimum(x int, ys ...int) int {  
    for _, y := range ys {  
        if y < x {  
            x = y  
        }  
    }  
    return x  
}  