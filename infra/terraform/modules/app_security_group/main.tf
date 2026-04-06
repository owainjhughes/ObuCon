# Adoption-safe security group for the EC2 app host.
#
# lifecycle ignore_changes on ingress/egress means Terraform manages only
# the group container (name, description, VPC).  The individual rules below
# are tracked as separate resources; any extra manually-added rules (e.g.
# SSH from personal IPs) are left untouched.

resource "aws_security_group" "app" {
  count = var.create_sg ? 1 : 0

  name        = var.sg_name
  description = "${var.sg_name} created by launch wizard"
  vpc_id      = var.vpc_id

  tags = var.tags

  lifecycle {
    ignore_changes = [ingress, egress, description, name, tags]
  }
}

# Ingress rules

resource "aws_vpc_security_group_ingress_rule" "ssh" {
  count = var.create_sg ? 1 : 0

  security_group_id = aws_security_group.app[0].id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "tcp"
  from_port         = 22
  to_port           = 22
  description       = "SSH"
}

resource "aws_vpc_security_group_ingress_rule" "http" {
  count = var.create_sg ? 1 : 0

  security_group_id = aws_security_group.app[0].id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "tcp"
  from_port         = 80
  to_port           = 80
  description       = "HTTP"
}

resource "aws_vpc_security_group_ingress_rule" "app_api" {
  count = var.create_sg ? 1 : 0

  security_group_id = aws_security_group.app[0].id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "tcp"
  from_port         = 8080
  to_port           = 8080
  description       = "App API port"
}

# Egress rules 

resource "aws_vpc_security_group_egress_rule" "all_outbound" {
  count = var.create_sg ? 1 : 0

  security_group_id = aws_security_group.app[0].id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
  description       = "Allow all outbound traffic"
}

resource "aws_vpc_security_group_egress_rule" "postgres_to_rds" {
  count = var.create_sg ? 1 : 0

  security_group_id            = aws_security_group.app[0].id
  referenced_security_group_id = var.rds_security_group_id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Allow outbound PostgreSQL to RDS"
}
