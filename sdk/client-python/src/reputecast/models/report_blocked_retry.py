from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_blocked_retry_unavailable import ReportBlockedRetryUnavailable
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_campaign_ref import ReportCampaignRef
  from ..models.report_retry_campaign import ReportRetryCampaign





T = TypeVar("T", bound="ReportBlockedRetry")



@_attrs_define
class ReportBlockedRetry:
    """ 
        Attributes:
            blocked (int):
            recipients (int):
            campaign (ReportRetryCampaign | Unset):
            of (ReportCampaignRef | Unset):
            unavailable (ReportBlockedRetryUnavailable | Unset):
     """

    blocked: int
    recipients: int
    campaign: ReportRetryCampaign | Unset = UNSET
    of: ReportCampaignRef | Unset = UNSET
    unavailable: ReportBlockedRetryUnavailable | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_campaign_ref import ReportCampaignRef # noqa: PLC0415
        from ..models.report_retry_campaign import ReportRetryCampaign # noqa: PLC0415
        blocked = self.blocked

        recipients = self.recipients

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        of: dict[str, Any] | Unset = UNSET
        if not isinstance(self.of, Unset):
            of = self.of.to_dict()

        unavailable: str | Unset = UNSET
        if not isinstance(self.unavailable, Unset):
            unavailable = self.unavailable.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocked": blocked,
            "recipients": recipients,
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if of is not UNSET:
            field_dict["of"] = of
        if unavailable is not UNSET:
            field_dict["unavailable"] = unavailable

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_campaign_ref import ReportCampaignRef # noqa: PLC0415
        from ..models.report_retry_campaign import ReportRetryCampaign # noqa: PLC0415
        d = dict(src_dict)
        blocked = d.pop("blocked")

        recipients = d.pop("recipients")

        _campaign = d.pop("campaign", UNSET)
        campaign: ReportRetryCampaign | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = ReportRetryCampaign.from_dict(_campaign)




        _of = d.pop("of", UNSET)
        of: ReportCampaignRef | Unset
        if isinstance(_of,  Unset):
            of = UNSET
        else:
            of = ReportCampaignRef.from_dict(_of)




        _unavailable = d.pop("unavailable", UNSET)
        unavailable: ReportBlockedRetryUnavailable | Unset
        if isinstance(_unavailable,  Unset):
            unavailable = UNSET
        else:
            unavailable = ReportBlockedRetryUnavailable(_unavailable)




        report_blocked_retry = cls(
            blocked=blocked,
            recipients=recipients,
            campaign=campaign,
            of=of,
            unavailable=unavailable,
        )

        return report_blocked_retry

