import { pathToFileURL } from "node:url";
import { join } from "node:path";
import { writeFile } from "node:fs/promises";

const [root, project, port, database] = process.argv.slice(2);
const file = (path) => pathToFileURL(join(root, path)).href;
const { seedVm, startGateway } = await import(file("test/harness.mjs"));
seedVm({ project });
const backup = seedVm({ project, id: "vm-sim-02" });
backup.claude.account_uuid = "acct-backup";
await writeFile(
	join(project, "vms", backup.id + ".json"),
	JSON.stringify(backup),
);
const gateway = await startGateway({
	project,
	port: Number(port),
	operations: false,
	env: {
		VM2API_DATABASE_URL: database,
		KIN_POSTGRES_TEST_URL: database,
		VM2API_DB_SECRET: "isolated-fixture",
		KIN_TEST_MODELS_DEV: "1",
		NODE_OPTIONS:
			"--import=" + file("test/fixtures/postgresql-proxy-pool-gateway.mjs"),
	},
});
process.once("SIGTERM", async () => {
	await gateway.stop();
	process.exit();
});
console.log("ready");
