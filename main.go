package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

var (
	app             = kingpin.New("Task", "Task management").DefaultEnvars()
	file            = app.Flag("file", "Filename of the tasks.").Required().String()
	showDone        = app.Flag("show-done", "Show tasks marked as done.").Short('d').Bool()
	exportFormat    = app.Flag("format", "Output format").Short('f').Default("table").Enum("table", "json")
	initFile        = app.Command("init", "Initialize the task file")
	stats           = app.Command("stats", "Show a bunch of statistics about the tasks")
	show            = app.Command("show", "Show tasks")
	showName        = show.Arg("name", "Task name").String()
	search          = app.Command("search", "Search for tasks").Alias("find")
	searchName      = search.Arg("string", "Part of task name or title").String()
	create          = app.Command("create", "Create task")
	createName      = create.Arg("name", "Task name").Required().String()
	createTitle     = create.Arg("title", "Task title").Required().Strings()
	deleteT         = app.Command("delete", "Delete task")
	deleteTName     = deleteT.Arg("name", "Task name").Required().String()
	setState        = app.Command("set-state", "Set task state")
	setStateName    = setState.Arg("name", "Task name").Required().String()
	setStateState   = setState.Arg("state", "State").Required().Enum("todo", "in-progress", "done")
	assign          = app.Command("assign", "Set task assignee")
	assignName      = assign.Arg("name", "Task name").Required().String()
	assignAssignee  = assign.Arg("state", "Assignee - you can use 'me', 'none' or empty (= 'me')").String()
	comment         = app.Command("comment", "Add a comment")
	commentName     = comment.Arg("name", "Task name").Required().String()
	commentComment  = comment.Arg("the comment", "The comment").Required().Strings()
	setField        = app.Command("set", "Set a custom field")
	setFieldName    = setField.Arg("name", "Task name").Required().String()
	setFieldFName   = setField.Arg("field-name", "Field name").Required().String()
	setFieldFValue  = setField.Arg("field value", "Field name").Required().Strings()
	unsetField      = app.Command("unset", "Set a custom field")
	unsetFieldName  = unsetField.Arg("name", "Task name").Required().String()
	unsetFieldFName = unsetField.Arg("field-name", "Field name").Required().String()

	lockfile string
)

func main() {
	var conf TaskConfig
	var err error
	var command string

	command = kingpin.MustParse(app.Parse(os.Args[1:]))
	lockfile = *file + ".lock"

	err = Lock(lockfile)
	if err != nil {
		panic(err)
	}
	defer Unlock(lockfile)

	if command == "init" {
		initTaskFile(*file)
	}

	if conf, err = readTasks(*file); err != nil {
		panic(err)
	}

	switch command {
	case "init":
	case "show":
		if *showName == "" {
			showTasks(&conf)
		} else {
			showTask(*showName, conf.Tasks[*showName])
			showTaskComments(*showName, conf.Tasks[*showName])
		}
	case "stats":
		showStats(&conf)
	case "search":
		searchTasks(*file, *searchName)
	case "create":
		createTask(*file, *createName, *createTitle)
	case "delete":
		deleteTask(*file, *deleteTName)
	case "set-state":
		setTaskState(*file, *setStateName, *setStateState)
	case "assign":
		setTaskAssignee(*file, *assignName, *assignAssignee)
	case "comment":
		addTaskComment(*file, *commentName, *commentComment)
	case "set":
		setTaskField(*file, *setFieldName, *setFieldFName, *setFieldFValue)
	case "unset":
		unsetTaskField(*file, *unsetFieldName, *unsetFieldFName)
	default:
		panic("Unknown command: " + command)
	}
}

func showTasks(conf *TaskConfig) {
	showSomeTasks(&conf.Tasks)
}

func parseUser(user string) string {
	if user == "none" {
		return ""
	}
	if user != "me" && user != "" {
		return user
	}
	if os.Getenv("SUDO_USER") != "" {
		return os.Getenv("SUDO_USER")
	} else {
		return os.Getenv("USER")
	}
}

func showSomeTasks(tasks *map[string]Task) {
	switch *exportFormat {
	case "table":
		showSomeTasksTable(tasks)
	case "json":
		showSomeTasksJson(tasks)
	}
}

func showSomeTasksJson(tasks *map[string]Task) {
	res, _ := json.Marshal(tasks)
	fmt.Println(string(res))
}

func showSomeTasksTable(tasks *map[string]Task) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Title", "State", "Assignee", "Comments"})
	for key, v := range *tasks {
		if *showDone || v.State != "done" {
			table.Append([]string{key, v.Title, v.State, v.Assignee, strconv.Itoa(len(v.Comments))})
		}
	}
	table.Render()
}

func showTask(name string, task Task) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT) // Set Alignment
	table.SetHeader([]string{"", "Showing task '" + name + "'"})
	table.Append([]string{"Title", task.Title})
	table.Append([]string{"Assignee", task.Assignee})
	table.Append([]string{"State", task.State})
	table.Append([]string{"Comments", strconv.Itoa(len(task.Comments))})
	table.Append([]string{"Created at", task.HumanCreatedAt()})
	table.Append([]string{"Updated at", task.HumanUpdatedAt()})
	for key, value := range task.Fields {
		table.Append([]string{key, value})
	}
	table.Render()
}

func showTaskComments(name string, task Task) {
	if len(task.Comments) > 0 {
		comments := tablewriter.NewWriter(os.Stdout)
		comments.SetHeader([]string{"", "Comments for task '" + name + "'", ""})
		for _, comment := range task.Comments {
			comments.Append([]string{comment.By, comment.Comment, comment.HumanAt()})
		}
		comments.Render()
	}
}

func showStats(conf *TaskConfig) {
	totalTasks := len(conf.Tasks)
	states := map[string]int{}
	var key string
	for _, task := range conf.Tasks {
		key = task.State
		if key == "" {
			key = "(unset)"
		}
		states[key] = states[key] + 1
	}

	for key, value := range states {
		fmt.Printf("%10s: %10d/%-10d (%.2f%%)\n", key, value, totalTasks, 100*float64(value)/float64(totalTasks))
	}
}

func searchTasks(file string, name string) {
	tasks := map[string]Task{}
	conf, _ := readTasks(file)

	for id, task := range conf.Tasks {
		if strings.Contains(id, name) ||
			strings.Contains(task.Title, name) {
			tasks[id] = task
		}
	}

	showSomeTasks(&tasks)
}

func deleteTask(file string, name string) {
	conf, _ := readTasks(file)
	if conf.Tasks != nil {
		if _, ok := conf.Tasks[name]; ok {
			delete(conf.Tasks, name)
			print("Deleted task '" + name + "'\n")
		} else {
			print("No task '" + name + "' found\n")
		}
	} else {
		print("No task '" + name + "' found\n")
	}
	writeTasks(file, &conf)
}
func createTask(file string, name string, titleArray []string) {
	title := strings.Join(titleArray, " ")
	conf, _ := readTasks(file)
	task := Task{
		Title: title,
	}
	task.Update()
	conf.Tasks[name] = task
	writeTasks(file, &conf)
	showTask(name, task)
}
func setTaskState(file string, name string, state string) {
	conf, _ := readTasks(file)
	task := conf.Tasks[name]
	task.State = state
	task.Update()
	conf.Tasks[name] = task
	writeTasks(file, &conf)
	showTask(name, task)
}

func setTaskAssignee(file string, name string, assignee string) {
	assignee = parseUser(assignee)
	conf, _ := readTasks(file)
	task := conf.Tasks[name]
	task.Assignee = assignee
	task.Update()
	conf.Tasks[name] = task
	writeTasks(file, &conf)
	showTask(name, task)
}

func addTaskComment(file string, name string, commentArray []string) {
	comment := strings.Join(commentArray, " ")
	user := parseUser("me")
	commentObj := TaskComment{
		Comment: comment,
		By:      user,
		At:      time.Now().Format(time.RFC3339),
	}
	conf, _ := readTasks(file)
	task := conf.Tasks[name]
	task.Comments = append(task.Comments, commentObj)
	task.Update()
	conf.Tasks[name] = task
	writeTasks(file, &conf)
	showTask(name, task)
}

func setTaskField(file string, name string, fieldName string, fieldValueArray []string) {
	fieldValue := strings.Join(fieldValueArray, " ")
	if fieldValue == "" {
		unsetTaskField(file, name, fieldName)
		return
	}
	conf, _ := readTasks(file)
	task := conf.Tasks[name]
	if task.Fields == nil {
		task.Fields = map[string]string{
			fieldName: fieldValue,
		}
	} else {
		task.Fields[fieldName] = fieldValue
	}
	task.Update()
	conf.Tasks[name] = task
	writeTasks(file, &conf)
	showTask(name, task)
}

func unsetTaskField(file string, name string, fieldName string) {
	conf, _ := readTasks(file)
	task := conf.Tasks[name]
	if task.Fields != nil {
		if _, ok := task.Fields[fieldName]; ok {
			delete(task.Fields, fieldName)
			print("Deleted field '" + fieldName + "' for task '" + name + "'\n")
			task.Update()
			conf.Tasks[name] = task
			writeTasks(file, &conf)
		} else {
			print("No field '" + fieldName + "' found for task '" + name + "'\n")
		}
	} else {
		print("No field '" + fieldName + "' found for task '" + name + "'\n")
	}
	showTask(name, task)
}

func writeTasks(file string, conf *TaskConfig) error {
	var err error
	var d []byte
	d, err = yaml.Marshal(&conf)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, d, 0644)
	return err
}

func readTasks(file string) (TaskConfig, error) {
	var conf TaskConfig
	var err error
	var dat []byte

	if dat, err = ioutil.ReadFile(file); err != nil {
		return conf, err
	}
	if err = yaml.Unmarshal(dat, &conf); err != nil {
		panic(err)
	}

	if conf.Tasks == nil {
		conf.Tasks = map[string]Task{}
	}

	return conf, nil
}
func initTaskFile(file string) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		var conf TaskConfig
		if err := writeTasks(file, &conf); err != nil {
			panic(err)
		}
	} else {
		print("File '" + file + "' already exists!\n")
	}
}

func Lock(file string) error {
	msg := false
	for {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			err := os.Mkdir(file, 0700)
			if !os.IsExist(err) {
				return err
			}
		}
		if !msg {
			print("Someone has a lock; waiting...\n")
			msg = true
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func Unlock(file string) error {
	return os.Remove(file)
}
