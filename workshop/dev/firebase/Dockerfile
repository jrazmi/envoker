FROM node:20-alpine

# install java
RUN apk add --no-cache openjdk11 bash curl

# Install Firebase CLI
RUN npm install -g firebase-tools

# Create a directory for Firebase emulator configuration
WORKDIR /app
# Create directory for emulator data
RUN mkdir -p /app/data

# Copy Firebase configuration files
COPY firebase.json .
COPY .firebaserc .


# Expose necessary ports
# Auth emulator
EXPOSE 9099
# Firestore emulator
# EXPOSE 8080
# Database emulator
# EXPOSE 9000
# Functions emulator
# EXPOSE 5001
# Hosting emulator
# EXPOSE 5000
# Pub/Sub emulator
# EXPOSE 8085
# Storage emulator
# EXPOSE 9199
# UI
EXPOSE 4000


# Command to run Firebase emulator
CMD ["firebase", "emulators:start", "--import=/app/data/export", "--export-on-exit=/app/data/export", "--project", "listening-post"]