package acceptance_test

import (
	"fmt"
	"os/exec"
	"time"

	acceptance "github.com/cloudfoundry/bosh-bootloader/acceptance-tests"
	"github.com/cloudfoundry/bosh-bootloader/acceptance-tests/actors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("up", func() {
	var (
		bbl             actors.BBL
		boshcli         actors.BOSHCLI
		directorAddress string
		caCertPath      string
		sshSession      *gexec.Session
		stateDir        string
		iaas            string
		iaasHelper      actors.IAASLBHelper
	)

	BeforeEach(func() {
		acceptance.SkipUnless("bbl-up")

		configuration, err := acceptance.LoadConfig()
		Expect(err).NotTo(HaveOccurred())

		iaas = configuration.IAAS
		iaasHelper = actors.NewIAASLBHelper(iaas, configuration)
		stateDir = configuration.StateFileDir

		bbl = actors.NewBBL(stateDir, pathToBBL, configuration, "up-env")
		boshcli = actors.NewBOSHCLI()
	})

	AfterEach(func() {
		if sshSession != nil {
			sshSession.Interrupt()
			Eventually(sshSession, "5s").Should(gexec.Exit())
		}

		session := bbl.Down()
		Eventually(session, 10*time.Minute).Should(gexec.Exit())
	})

	It("bbl's up a new bosh director and jumpbox", func() {
		args := []string{
			"--name", bbl.PredefinedEnvID(),
		}
		args = append(args, iaasHelper.GetLBArgs()...)
		session := bbl.Up(args...)
		Eventually(session, 40*time.Minute).Should(gexec.Exit(0))

		By("creating an ssh tunnel to the director in print-env", func() {
			sshSession = bbl.StartSSHTunnel()
		})

		By("checking if the bosh director exists", func() {
			directorAddress = bbl.DirectorAddress()
			caCertPath = bbl.SaveDirectorCA()

			directorExists := func() bool {
				exists, err := boshcli.DirectorExists(directorAddress, caCertPath)
				if err != nil {
					fmt.Println(string(err.(*exec.ExitError).Stderr))
				}
				return exists
			}
			Eventually(directorExists, "1m", "10s").Should(BeTrue())
		})

		By("checking that the cloud config exists", func() {
			directorUsername := bbl.DirectorUsername()
			directorPassword := bbl.DirectorPassword()

			cloudConfig, err := boshcli.CloudConfig(directorAddress, caCertPath, directorUsername, directorPassword)
			Expect(err).NotTo(HaveOccurred())
			Expect(cloudConfig).NotTo(BeEmpty())
		})

		By("checking if bbl print-env prints the bosh environment variables", func() {
			stdout := bbl.PrintEnv()

			Expect(stdout).To(ContainSubstring("export BOSH_ENVIRONMENT="))
			Expect(stdout).To(ContainSubstring("export BOSH_CLIENT="))
			Expect(stdout).To(ContainSubstring("export BOSH_CLIENT_SECRET="))
			Expect(stdout).To(ContainSubstring("export BOSH_CA_CERT="))
		})

		By("rotating the jumpbox's ssh key", func() {
			sshKey := bbl.SSHKey()
			Expect(sshKey).NotTo(BeEmpty())

			session = bbl.Rotate()
			Eventually(session, 40*time.Minute).Should(gexec.Exit(0))

			rotatedKey := bbl.SSHKey()
			Expect(rotatedKey).NotTo(BeEmpty())
			Expect(rotatedKey).NotTo(Equal(sshKey))
		})

		By("checking bbl up is idempotent", func() {
			session := bbl.Up()
			Eventually(session, 40*time.Minute).Should(gexec.Exit(0))
		})

		By("confirming that the load balancers exist", func() {
			iaasHelper.ConfirmLBsExist(bbl.PredefinedEnvID())
		})

		By("verifying that vm extensions were added to the cloud config", func() {
			cloudConfig := bbl.CloudConfig()
			vmExtensions := acceptance.VmExtensionNames(cloudConfig)
			Expect(vmExtensions).To(ContainElement("cf-router-network-properties"))
			Expect(vmExtensions).To(ContainElement("diego-ssh-proxy-network-properties"))
			Expect(vmExtensions).To(ContainElement("cf-tcp-router-network-properties"))
		})

		By("verifying the bbl lbs output", func() {
			stdout := bbl.Lbs()
			Expect(stdout).To(MatchRegexp("CF Router LB:.*"))
			Expect(stdout).To(MatchRegexp("CF SSH Proxy LB:.*"))
			Expect(stdout).To(MatchRegexp("CF TCP Router LB:.*"))
		})

		By("deleting lbs", func() {
			session := bbl.Plan("--name", bbl.PredefinedEnvID())
			Eventually(session, 1*time.Minute).Should(gexec.Exit(0))

			session = bbl.Up()
			Eventually(session, 40*time.Minute).Should(gexec.Exit(0))
		})

		By("confirming that the load balancers no longer exist", func() {
			iaasHelper.ConfirmNoLBsExist(bbl.PredefinedEnvID())
		})

		By("destroying the director and the jumpbox", func() {
			session := bbl.Down()
			Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
		})
	})
})
