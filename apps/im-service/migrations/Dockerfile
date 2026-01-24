# Custom Liquibase image with MySQL JDBC driver
FROM liquibase/liquibase:4.25

# Download and install MySQL Connector/J
USER root
RUN lpm add mysql --global
USER liquibase

# Set working directory
WORKDIR /liquibase
