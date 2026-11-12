from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.overview_next_issue import OverviewNextIssue
  from ..models.overview_scheduled_campaign import OverviewScheduledCampaign





T = TypeVar("T", bound="OverviewNextCampaign")



@_attrs_define
class OverviewNextCampaign:
    """ 
        Attributes:
            drafts (int):
            campaign (OverviewScheduledCampaign | Unset):
            issue (OverviewNextIssue | Unset):
     """

    drafts: int
    campaign: OverviewScheduledCampaign | Unset = UNSET
    issue: OverviewNextIssue | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_next_issue import OverviewNextIssue # noqa: PLC0415
        from ..models.overview_scheduled_campaign import OverviewScheduledCampaign # noqa: PLC0415
        drafts = self.drafts

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        issue: dict[str, Any] | Unset = UNSET
        if not isinstance(self.issue, Unset):
            issue = self.issue.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "drafts": drafts,
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if issue is not UNSET:
            field_dict["issue"] = issue

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_next_issue import OverviewNextIssue # noqa: PLC0415
        from ..models.overview_scheduled_campaign import OverviewScheduledCampaign # noqa: PLC0415
        d = dict(src_dict)
        drafts = d.pop("drafts")

        _campaign = d.pop("campaign", UNSET)
        campaign: OverviewScheduledCampaign | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = OverviewScheduledCampaign.from_dict(_campaign)




        _issue = d.pop("issue", UNSET)
        issue: OverviewNextIssue | Unset
        if isinstance(_issue,  Unset):
            issue = UNSET
        else:
            issue = OverviewNextIssue.from_dict(_issue)




        overview_next_campaign = cls(
            drafts=drafts,
            campaign=campaign,
            issue=issue,
        )

        return overview_next_campaign

