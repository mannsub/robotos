// Jenkinsfile
pipeline {
    agent any

    options {
        timeout(time: 30, unit: 'MINUTES')
        disableConcurrentBuilds()
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Test: Go') {
            steps {
                dir('middleware') {
                    sh '''
                        export PATH=$PATH:/usr/local/go/bin
                        go test ./... -v -race
                    '''
                }
            }
        }

        stage('Test: Python NeoDM') {
            steps {
                dir('middleware/services/neodm') {
                    sh '''
                        python3 -m venv .venv
                        . .venv/bin/activate
                        pip install -q -r requirements.txt
                        python3 -m pytest test_neodm.py -v
                    '''
                }
            }
        }

        stage('Build: Go Image') {
            steps {
                sh 'docker build -f docker/go.Dockerfile -t robotos:ci-${BUILD_NUMBER} .'
            }
        }

        stage('Build: NeoDM Image') {
            steps {
                sh 'docker build -f docker/neodm.Dockerfile -t neodm:ci-${BUILD_NUMBER} .'
            }
        }
    }

    post {
        success {
            withCredentials([string(credentialsId: 'slack-webhook', variable: 'WEBHOOK')]) {
                sh """
                    curl -s -X POST -H 'Content-type: application/json' \
                    --data '{"text":"*✅ Build Passed* — ${env.JOB_NAME} #${env.BUILD_NUMBER}\\nBranch: ${env.GIT_BRANCH}\\n<${env.BUILD_URL}|View Build>"}' \
                    \$WEBHOOK
                """
            }
        }
        failure {
            withCredentials([string(credentialsId: 'slack-webhook', variable: 'WEBHOOK')]) {
                sh """
                    curl -s -X POST -H 'Content-type: application/json' \
                    --data '{"text":"*❌ Build Failed* — ${env.JOB_NAME} #${env.BUILD_NUMBER}\\nBranch: ${env.GIT_BRANCH}\\n<${env.BUILD_URL}|View Build>"}' \
                    \$WEBHOOK
                """
            }
        }
        always {
            cleanWs()
        }
    }
}