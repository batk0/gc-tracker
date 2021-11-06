runtime: go116

instance_class: F1

env_variables:
  PROJECT_NAME: ${{ secrets.PROJECT_NAME }}
  SESSION_COLLECTION: ${{ secrets.SESSION_COLLECTION }}
  SMTP_HOST: ${{ secrets.SMTP_HOST }}
  SMTP_PORT: ${{ secrets.SMTP_PORT }}
  SMTP_USER: ${{ secrets.SMTP_USER }}
  SMTP_PASS: ${{ secrets.SMTP_PASS }}
