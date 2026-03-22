pub(crate) const MAKE_CHECK: &str = "make check";
pub(crate) const MAKE_VERIFY: &str = "make verify";
pub(crate) const MAKE_SMOKE: &str = "make smoke";

pub(crate) const CHECK_REPO: &str = "raccoon-cli check repo";
pub(crate) const CHECK_TOPOLOGY: &str = "raccoon-cli check topology";
pub(crate) const CHECK_CONTRACTS: &str = "raccoon-cli check contracts";
pub(crate) const CHECK_BINDINGS: &str = "raccoon-cli check bindings";
pub(crate) const CHECK_ARCH: &str = "raccoon-cli check arch";
pub(crate) const CHECK_DRIFT: &str = "raccoon-cli check drift";

pub(crate) const INSPECT_COVERAGE: &str = "raccoon-cli inspect coverage";

pub(crate) const CHANGE_IMPACT: &str = "raccoon-cli change impact";
pub(crate) const CHANGE_TDD: &str = "raccoon-cli change tdd";
pub(crate) const CHANGE_BRIEFING: &str = "raccoon-cli change briefing";
pub(crate) const CHANGE_RECOMMEND: &str = "raccoon-cli change recommend";

pub(crate) fn check_gate(profile: &str) -> String {
    format!("raccoon-cli check gate --profile {profile}")
}
