## *Web Application for determining the suitability of online resources for immersion language-learners*

### Project description: 
Online web application that determines the suitability of an online text-based resource, such as a short story or newspaper article, for people learning a non-native language. It determines suitability of a resource two ways: Through the grade/school level of individual words, and by analyzing the words present and comparing them to the words the user has already learned; which would be progressively updated as the user uses the software more. Initial implementation will be focused on Japanese; More langauages will be implemented further down the line. (Name Idea: ObuSearch. Obu = Parrot, known for intelligence, mimicking, learning and communication, which is what the app is about, or 'ObuCon' Where Con means to learn)

It is like a much more personalised and learning-focused version of [jreadability](https://jreadability.net/sys/en) or [Kuma Learn](https://app.kumalearn.com/text-analysis)

Can be foudn at [this link](https://obucon.com)

#### Architecture Decisions

##### Stack Choices
- Frontend: React
- Backend: Go + Gin
- Database: PostgreSQL
- Deployment: Docker, Terraform, AWS (Cloudfront + EC2 + RDS)
- CI/CD: GitHub Actions

### How to use

In frontend:
```
npm run start
```

In backend:
```
go run cmd/server/main.go
```