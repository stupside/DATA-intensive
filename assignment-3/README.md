# Assignment 3

For this assignment, I have made a dummy course enrolment system. 

[Link to the video](https://youtu.be/zidvYysT-Vw)

## Build the project

I am using [go-task](https://github.com/go-task/task) to run build command.
If you don't want to use this tool, you can take a look at the `Taskfile.yml` and run the commands yourself.

## How to setup the db

Three MongoDB databases are used and you can use the following command to set it up with docker.
If you want more databases, or use the ones you already have, you can modify the `config.json` file.

```bash
docker compose up #-d
```

If you want to connect you own databases, make sure to update the `config.json` file.

## Some usefull commands

For every commands, you can run `--help` to get more informations about how to use the CLI.

```bash
# At any time you can see help for command and sub-commands.
go run cmd/main.go -- --help
go run cmd/main.go -- mock --help
go run cmd/main.go -- list --help
go run cmd/main.go -- list students --help
# ...

# The first step is to generate some data for the databases.
go run cmd/main.go -- mock gen

# The next step is to seed the databases using this generated data. 
go run cmd/main.go -- mock db seed

# Then you can start playing arround with the CLI.

go run cmd/main.go -- list students --db university_db1
go run cmd/main.go -- list professors --db university_db1
go run cmd/main.go -- list departments --db university_db1

go run cmd/main.go -- student info --db university_db1 --student-id stud_2
go run cmd/main.go -- department info --db university_db1 --department-id dept_1

go run cmd/main.go -- student enrollment add --db university_db1 --course-id course_1 --sid stud_2
go run cmd/main.go -- student enrollment remove --db university_db1 --course-id course_1 --sid stud_2

# Enroll student from one department to a course in another department
go run cmd/main.go -- student enrollment add --db university_db1 --course-id course_3 --sid stud_2
```