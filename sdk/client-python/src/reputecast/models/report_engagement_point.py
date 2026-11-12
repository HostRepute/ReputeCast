from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportEngagementPoint")



@_attrs_define
class ReportEngagementPoint:
    """ 
        Attributes:
            clicks (int):
            hour (int):
            human_opens (int):
            machine_opens (int):
            privacy_opens (int):
     """

    clicks: int
    hour: int
    human_opens: int
    machine_opens: int
    privacy_opens: int





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        hour = self.hour

        human_opens = self.human_opens

        machine_opens = self.machine_opens

        privacy_opens = self.privacy_opens


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "hour": hour,
            "humanOpens": human_opens,
            "machineOpens": machine_opens,
            "privacyOpens": privacy_opens,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        hour = d.pop("hour")

        human_opens = d.pop("humanOpens")

        machine_opens = d.pop("machineOpens")

        privacy_opens = d.pop("privacyOpens")

        report_engagement_point = cls(
            clicks=clicks,
            hour=hour,
            human_opens=human_opens,
            machine_opens=machine_opens,
            privacy_opens=privacy_opens,
        )

        return report_engagement_point

