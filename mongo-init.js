db.createUser(
  {
    user: "anynines",
    pwd: "password",
    roles: [
      {
      role: "readWrite",
      db: "mongodb"
      }
    ]
  }
);
