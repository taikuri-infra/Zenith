# =============================================================================
# Zenith Auth Secrets — SealedSecret for email verification + OAuth
# =============================================================================
# This SealedSecret is decrypted by the Sealed Secrets controller into a
# regular K8s Secret "zenith-auth-secrets" in the platform namespace.
#
# To re-seal after rotating keys:
#   KUBECONFIG=~/.kube/zenith-staging.yaml kubeseal --fetch-cert \
#     --controller-namespace sealed-secrets --controller-name sealed-secrets > /tmp/cert.pem
#
#   cat <<EOF | kubeseal --cert /tmp/cert.pem --format yaml
#   { "apiVersion": "v1", "kind": "Secret", "metadata": { "name": "zenith-auth-secrets",
#     "namespace": "zenith-staging" }, "type": "Opaque",
#     "stringData": { "resend-api-key": "re_xxx", "google-client-id": "xxx.apps.googleusercontent.com",
#       "google-client-secret": "GOCSPX-xxx", "github-client-id": "Ov23xxx", "github-client-secret": "xxx" } }
#   EOF

resource "kubernetes_manifest" "zenith_auth_sealed_secret" {
  count = var.enable_sealed_secrets ? 1 : 0

  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "bitnami.com/v1alpha1"
    kind       = "SealedSecret"
    metadata = {
      name      = "zenith-auth-secrets"
      namespace = var.platform_namespace
    }
    spec = {
      encryptedData = {
        "resend-api-key"       = "AgC2CCHuMpnYvinrM43ng4syid5Es9TZKpd5IRprVczzBUBQMlaiIRPvjsEWdFv0YXKn8f/U5YRQ8APUBnAvNISjODxQClvC07ZMo2hN9yiFhbwBTkaDW4zQbLf/F5byRc9xDKQp0AIPrHQu0UNTXkZkUMlv0V3fCC00fCFPt2MSjaQpVVOAWCEZN5m22I2ST1Lm8aEEdK/8JUsIs2OxZwwnmbnWcLfqFxEF/ggxqu3973F29iPWOUObdGRLTFpSmVKsu7nESwIm0JYHx+Y6HZbD6OI0rAYgi7646zBbwbM66ud+rQCfhXHO9+nisXOzKKbGAs6CbtTy2kr7wkuBV4Z1Y8E4x9EM4fBS86A8d6kdBc239bGUOh6u9+yORGBjWL2ufoquGHhlH5zwNsfR1MBXhlZWwTjL6OEW0YPUjNcM/T4OkkIlHWsfVyCDTSyNjSZxlCCV7SCNtWrZV5LUrRCdRdptEHuEtECklx5oBJzKS9XZQaeAr5PA/cTlSt6pRx2t0gv/lhIi8u0CD1v4aXPGE5BoAYH1Ys9oKhEcwOVP1FT6/X7pY7OIsNVd1JUO41B+ef/tqDx+XWhwjOvr2fjQ74H5Z1MOrRgoKt14bc6YlKKckA0NZhIEvsMMIsGjX89XDY3lzoU6miJd0WIBcWQ1TekXZ2KkYm3erBQ5T0MPaUsEUL9laTEol1pNXIz/1QrGV7xd23grySSB7Z+c14ymhU7JKvSLs08Mov5oWyzd5vQ1/u0="
        "google-client-id"     = "AgCylkuhtCiVZ9alV+jYzP20pJO4Y9vc5+wBIxB2RgjLr0rO4E9ydJ6ckjNAlGJP1K0Us9QBE8ie38o3QD5vDx7KFNOYdFVPA4mTN+8vLadg1iTvQMNJbDs9h/7vFvHTyJxUGOfc9vKEmh9SYixAcZzerMZDvjECRHzxGNeZEgGE5hUy10jiazAYPtJmh/MvCfsNCtQMVhn9ZXpWxaKGLm4dD559mcDzoIzU3EsFNo1j81Ndx230VRGarRJ90ADc8eRQ+nF0rX2Jt7uKGrSFeHwxvNuzWpzfl2kaEHu9/5/rtVLGxHgRzrx5pyoeK0bCk2VNmJRmv95re2ja3WOTl754CguKue3ioHqZgkgttOfFzXEd389NODJZUGlFXMF8az527WfXdYn31pkxCCYE1YfnbohEHSScxM6NapvpRPCefOdCy8tGpl1jcsV+Etlm3SUtdr4F3s/GCXKG/FBiqxQhL3ZO8cE8goTggsHQK2wUKk6NSvuDGU+qn1DhHF6e57V35Vv5ziSgPhlXLWaAC1xCXAFd7ZM4/EgtW8mN+K5bx8xkQkBjUI6rEKqQimi0Yi+9lKpr3fmGVf+Ds2fzh2FMcorFgeSpkUkz/B1gZVHfsGaKHH5wwQ9f3QcVEflPVF+cRCRAYCk3XJ3WCAPVhYr8SBhS+oeyjYpO99eIdtfnalRSA1X66kcLl3/nN9uhRSw9VJAtpTvNrO3HYGNYYNo6rzTcetptTdsKSAbxeHEOsXRVu/qh1jA6DCF98I/MomcRbHqGKlcX1xjcPrbdxdf9sogW9I7ZPA=="
        "google-client-secret" = "AgAYYRMnubws4GKodJ76s2JmWr4+8Suq4YEFLb/RDMEWkZkWVSLLLZcxRIWFmEahfK83HcABmg6yY4IxHf2VOE6Js06+dWXgzjk3/xew666WINp6+wIsCgjlSMP58wpEZb9Cr1Wwnu+C9EKz42TWdmHUDU3j1le/MU8qIm0Eqlq3OFr8rMxfFtf+m+2lMY8MLoUddUglFrIoqXAt+SeOw1eeyYKcBICiHzgCY8Nm7uKpQhHUSr2KG73wXOANOAKhnDjD11IlXAwxA9IbJ1J3YZs6vqj53rM89fIgmpoYqN2IOswJJ6QTJmgXd8RvbyvIEfLvgR3yPRFYbOA/Dni7ScfMJoS777DBpJUWg3vRDKRsHX5KRLOoUDMsAXtzXbbTuW5mcVd9+a7+7mSmAYSIoRyFgkDE2Sa64RgfjjzRZ6Yi/rIOr/meIdGZFIoFNcChY2adMRN4L5A9+qSEFe0Dvrqb+hztC72eVTOLSEhnzKZz5X9Lf6tHKMyAp5b9ScgerU/7I3bpd2KbvCY1j/eY99zaP6twACBTep1fR/+xXXosPpp920OEjozAwVItqz8xBFPJw8vSn7FrSHjGWFmg64g+diWlUmgWO7QLapfMwbGLFr7SC53ZCheAnk++T1Fz+hBHOUhZd+K+3JkmNkKfF+7Mqb07IHJgJgLoKosBLJ+vHqzd8u6CHdm9wq1MbuxQk3xhJSiqrNBOSBC7KGNKYlh7fXQHCnZYCytZafpBucMMycAVaA=="
        "github-client-id"     = "AgClsmaV3lKb3P8jR+UsuGk4+2yqTNGG+gy+3eEMFi7fgO95fS2WxOKtl2n+akra5UAmR5RAVKmir96DALqcVpDHyuzfawcVAO8L47XWVmKEka5tKRpKxlJFmxxYLPu0fIrYIHj2DopeWoAGHGkG3UjvXNzaLeBKPPj5awNoI2F3sFb7Bq5MDT6syJb1KyZT45bVI0TkiyG2+kamfrhzMux8aqvFKToqsgJ9h5TH4/Znt7oUkyrYIVIoaHzSdN3T/rpeym1BzRbTC5tIIBhEjGo6rucw0HP4V6VvKHg1XnJMBSvKj3o2aw75egim1ot5YDobVqi1sZarjOK4hG+EiAXtq05kiXhdZQiCWE5Y3OAiSxaTl8pmmr1jdG7rrTCbsu22XKEW1P1+n/E6ou6UscEA5sdYc3DBo8LefzjSGNVEmr3agMKQvRjfoaCUnLq7ZDUQfZFnaVyiGbnHvBYTjesfONTYZq2lNzWhAO5lbZH9H5WSHN0VfuW8YNIn28XMNuH+s0hZMniMiGa1HOsr1gFsusDTr5Z96Vp+VZamdDZY0DVqny/KJ8/HwL08HWeKENxGvoqrz5/BZjdhDdaOSpmvH/NpP+ESGU8plREK6Zr9KguHM+9iwD1o/1Cooo3w73Dhc6jG5Wp2/jImEIKm83acW7fkSWpWKqV733j6n1leKaotq1fvkVEp3qWRYM6MrFKSMbbEeSWzQd1bsLSBYv4p26uIpg=="
        "github-client-secret" = "AgBFYtWfxzhuqDtfIdRFyZ7E1v0lED9gzhpECH6GKnwv+nSyhZHQ/r/cfOJdMQnEGAQKklUaAGuiUrtd6lNlYl+Lseg9ylx+GCPGNMwZAsnww3PVhRfXSJqPVonMYIRk0PRbN+Xe+BFAkKOVEazY7aNkNpNN8243KUrDiAOpCBWyoyOjP6Yfh0pEbTa+B3h/dlXPvzm4QTJ7aUEOCyoupphSC1eIDA9n+uKuOpaSP1/qrK6aSDMAjeLKasxPeH1E3PAAAQnlpif9osSM/xe5kz9M1bLmyJ0ZubQz3iCeBXIgRR3mHwKtbDJNzzE3ORRDCeIeyDwpfHqMGRSreAfByJ3xb9ptr7Xip8tuQFkHBoMSEQTJUZ8glJllsZzvJvLUNWboZPCcXwyGCIz/xijIra47RgzCDA3JIFEzTdMdGYGGt6Y3Dap76jbpxOLHFtFtXP81F0dfY28xSNiut+qSyMmk+qJkmCH74vkv2kKC8ol/d2pr+SqXFXcOxhaLJsVngdzrRveZles429FE+VJPWyW0CaQmJCfKfvPbFx37bCPXzIJxh6My/+z4JDPNzjWmt8hrwdIwgzlglQ1j7ItasL2Z2c5w3twIV44EKOGNGAvGQxOjW9a7noVHb4L40GNPsn57KKNmUhn8eTWDvVnWdDzvf6EfojeceAsnYD7MmymOkSG1aumCwFZnNh1iGRAgu0XSzOByXMJ1tGGujvYhLHZOuT0HvWmnTrtK0U0q3SNAT5xT4Oj2iwl+"
      }
      template = {
        metadata = {
          name      = "zenith-auth-secrets"
          namespace = var.platform_namespace
          labels = {
            "app.kubernetes.io/part-of"   = "zenith"
            "app.kubernetes.io/component" = "auth"
          }
        }
        type = "Opaque"
      }
    }
  }

  depends_on = [helm_release.sealed_secrets]
}
