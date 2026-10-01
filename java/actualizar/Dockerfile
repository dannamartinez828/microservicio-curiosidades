FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /src
COPY pom.xml .
COPY src ./src
RUN mvn -q package -DskipTests
FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /src/target/mascotas-service-1.0.jar app.jar
ENV OPERATION=actualizar
CMD ["java","-jar","app.jar"]
