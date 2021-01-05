@Library('dst-shared@master') _

dockerBuildPipeline {
   repository = "cray"
   imagePrefix = "hms"
   app = "hmcollector"
   name = "hms-hmcollector"
   description = "Collector for HMS Telemetry"
   dockerfile = "Dockerfile"
   slackNotification = ["", "", false, false, true, true]
   product = "csm"
}
