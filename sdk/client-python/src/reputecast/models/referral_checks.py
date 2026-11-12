from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.referral_campaign_ref import ReferralCampaignRef
  from ..models.referral_check import ReferralCheck





T = TypeVar("T", bound="ReferralChecks")



@_attrs_define
class ReferralChecks:
    """ 
        Attributes:
            checks (list[ReferralCheck]):
            links (int):
            pass_ (bool):
            next_issue (ReferralCampaignRef | Unset):
     """

    checks: list[ReferralCheck]
    links: int
    pass_: bool
    next_issue: ReferralCampaignRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_campaign_ref import ReferralCampaignRef # noqa: PLC0415
        from ..models.referral_check import ReferralCheck # noqa: PLC0415
        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        links = self.links

        pass_ = self.pass_

        next_issue: dict[str, Any] | Unset = UNSET
        if not isinstance(self.next_issue, Unset):
            next_issue = self.next_issue.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checks": checks,
            "links": links,
            "pass": pass_,
        })
        if next_issue is not UNSET:
            field_dict["nextIssue"] = next_issue

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_campaign_ref import ReferralCampaignRef # noqa: PLC0415
        from ..models.referral_check import ReferralCheck # noqa: PLC0415
        d = dict(src_dict)
        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = ReferralCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        links = d.pop("links")

        pass_ = d.pop("pass")

        _next_issue = d.pop("nextIssue", UNSET)
        next_issue: ReferralCampaignRef | Unset
        if isinstance(_next_issue,  Unset):
            next_issue = UNSET
        else:
            next_issue = ReferralCampaignRef.from_dict(_next_issue)




        referral_checks = cls(
            checks=checks,
            links=links,
            pass_=pass_,
            next_issue=next_issue,
        )

        return referral_checks

