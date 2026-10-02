FROM eclipse-temurin:21-jdk
WORKDIR /app
COPY lib ./lib
COPY Main.java .
RUN javac -cp "lib/*" -d out Main.java
EXPOSE 10000
CMD ["java", "-Xmx256m", "-XX:+UseSerialGC", "-cp", "out:lib/*", "Main"]
