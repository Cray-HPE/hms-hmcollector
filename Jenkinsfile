@Library('dst-shared@release/shasta-1.4') _

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
