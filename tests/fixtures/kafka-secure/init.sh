#!/usr/bin/env bash
set -euo pipefail

store_password="local-store-password"
key_password="${store_password}"
scram_username="transaction-local"
scram_password="local-scram-password"

keytool -genkeypair -noprompt -alias kafka-secure -keyalg RSA -keysize 2048 -validity 2 \
  -dname "CN=kafka-secure" -ext "SAN=dns:kafka-secure" \
  -keystore /secrets/kafka.keystore.jks -storepass "${store_password}" -keypass "${key_password}"
keytool -exportcert -rfc -alias kafka-secure -keystore /secrets/kafka.keystore.jks \
  -storepass "${store_password}" -file /secrets/kafka-ca.pem
keytool -importcert -noprompt -alias kafka-secure -file /secrets/kafka-ca.pem \
  -keystore /secrets/kafka.truststore.jks -storepass "${store_password}"

printf '%s' "${store_password}" > /secrets/kafka_keystore_creds
printf '%s' "${key_password}" > /secrets/kafka_ssl_key_creds
printf '%s' "${store_password}" > /secrets/kafka_truststore_creds
cat > /secrets/client.properties <<EOF
security.protocol=SASL_SSL
sasl.mechanism=SCRAM-SHA-512
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="${scram_username}" password="${scram_password}";
ssl.truststore.location=/etc/kafka/secrets/kafka.truststore.jks
ssl.truststore.password=${store_password}
ssl.endpoint.identification.algorithm=https
EOF
cat > /secrets/bad-client.properties <<EOF
security.protocol=SASL_SSL
sasl.mechanism=SCRAM-SHA-512
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="${scram_username}" password="wrong-password";
ssl.truststore.location=/etc/kafka/secrets/kafka.truststore.jks
ssl.truststore.password=${store_password}
ssl.endpoint.identification.algorithm=https
EOF
cat > /secrets/kafka_server_jaas.conf <<'EOF'
KafkaServer {
  org.apache.kafka.common.security.scram.ScramLoginModule required;
};
EOF

/opt/kafka/bin/kafka-storage.sh format --ignore-formatted \
  --cluster-id 4L6g3nShT-eMCtK--X86sw \
  --config /fixtures/server.properties \
  --add-scram "SCRAM-SHA-512=[name=${scram_username},password=${scram_password}]"
chown -R 1000:1000 /secrets /var/lib/kafka/data
chmod 0600 /secrets/*
chmod 0644 /secrets/kafka-ca.pem
