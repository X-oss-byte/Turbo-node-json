use turbo_tasks_build::{generate_register, rerun_if_glob};

fn main() {
    generate_register();
    // The test/snapshot crate need to be rebuilt if any snapshots are added.
    rerun_if_glob("tests/execution/*/*/*", "tests/execution");
    rerun_if_glob("tests/execution/*/*/__skipped__/*", "tests/execution");
    rerun_if_glob("tests/snapshot/*/*", "tests/snapshot");
}
