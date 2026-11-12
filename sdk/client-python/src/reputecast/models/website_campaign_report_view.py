from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.website_campaign_report import WebsiteCampaignReport





T = TypeVar("T", bound="WebsiteCampaignReportView")



@_attrs_define
class WebsiteCampaignReportView:
    """ 
        Attributes:
            report (WebsiteCampaignReport | Unset):
     """

    report: WebsiteCampaignReport | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_campaign_report import WebsiteCampaignReport # noqa: PLC0415
        report: dict[str, Any] | Unset = UNSET
        if not isinstance(self.report, Unset):
            report = self.report.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if report is not UNSET:
            field_dict["report"] = report

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_campaign_report import WebsiteCampaignReport # noqa: PLC0415
        d = dict(src_dict)
        _report = d.pop("report", UNSET)
        report: WebsiteCampaignReport | Unset
        if isinstance(_report,  Unset):
            report = UNSET
        else:
            report = WebsiteCampaignReport.from_dict(_report)




        website_campaign_report_view = cls(
            report=report,
        )

        return website_campaign_report_view

