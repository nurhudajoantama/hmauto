pipeline {
  agent any

  environment {
    BINARY_NAME = 'hmstt-linux'
    GOOS = 'linux'
    GOARCH = 'amd64'
    // These should be provided as Pipeline/Job environment variables or credentials in Jenkins
    // REMOTE_HOST, REMOTE_USER, REMOTE_PATH can be overridden in the job configuration
    REMOTE_HOST = "${env.REMOTE_HOST ?: 'your.remote.host'}"
    REMOTE_USER = "${env.REMOTE_USER ?: 'deploy'}"
    REMOTE_PATH = "${env.REMOTE_PATH ?: '/opt/stthmauto'}"
    SSH_CREDENTIALS_ID = "${env.SSH_CREDENTIALS_ID ?: 'deploy-ssh1'}"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Setup Go Environment') {
      steps {
        sh '''
          echo "Setting up Go environment"
          go version
          go mod tidy
        '''
      }
    }

    stage('Build') {
      steps {
        sh '''
          echo "Building ${BINARY_NAME} for ${GOOS}/${GOARCH}"
          # Build static linux binary
          CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags "-s -w" -o ${BINARY_NAME} .
          ls -lh ${BINARY_NAME}
        '''
        archiveArtifacts artifacts: "${BINARY_NAME}", fingerprint: true
        stash includes: "${BINARY_NAME}", name: 'binary'
      }
    }

    stage('Package Views') {
      steps {
        sh '''
          # Package the views folder (only the hmstt view)
          tar -czf views-hmstt.tar.gz -C views hmstt
          ls -lh views-hmstt.tar.gz
        '''
        archiveArtifacts artifacts: 'views-hmstt.tar.gz', fingerprint: true
        stash includes: 'views-hmstt.tar.gz', name: 'views'
      }
    }

    stage('Deploy') {
      steps {
        // Ensure we have the built artifacts available in the workspace
        unstash 'binary'
        unstash 'views'

        script {
          // Use the Jenkins SSH credentials (ssh-agent plugin) to copy files
          sshagent (credentials: [env.SSH_CREDENTIALS_ID]) {
            sh '''
              set -e
              echo "Copying ${BINARY_NAME} and views to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}"
              # Create remote path if missing and copy files
              scp -o StrictHostKeyChecking=no ${BINARY_NAME} ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}/
              scp -o StrictHostKeyChecking=no views-hmstt.tar.gz ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}/
              # Extract views on remote and set executable bit for binary
              ssh -o StrictHostKeyChecking=no ${REMOTE_USER}@${REMOTE_HOST} \
                "mkdir -p ${REMOTE_PATH} && cd ${REMOTE_PATH} && tar -xzf views-hmstt.tar.gz && chmod +x ${BINARY_NAME}"
            '''
          }
        }
      }
    }
  }

  post {
    always {
      cleanWs()
    }
  }
}