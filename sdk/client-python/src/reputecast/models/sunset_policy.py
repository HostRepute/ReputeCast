from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SunsetPolicy")



@_attrs_define
class SunsetPolicy:
    """ 
        Attributes:
            active_days (int):
            at_risk_days (int):
            count_machine_opens (bool):
            delete_after (bool):
            delete_after_months (int):
            enabled (bool):
            reengage (bool):
            stop_after_days (int):
            stop_broadcasts (bool):
            sunset_days (int):
            confirm_erase (str | Unset):
            enabled_at (str | Unset):
            reengage_automation (str | Unset): Name of the automation that sends it
     """

    active_days: int
    at_risk_days: int
    count_machine_opens: bool
    delete_after: bool
    delete_after_months: int
    enabled: bool
    reengage: bool
    stop_after_days: int
    stop_broadcasts: bool
    sunset_days: int
    confirm_erase: str | Unset = UNSET
    enabled_at: str | Unset = UNSET
    reengage_automation: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        active_days = self.active_days

        at_risk_days = self.at_risk_days

        count_machine_opens = self.count_machine_opens

        delete_after = self.delete_after

        delete_after_months = self.delete_after_months

        enabled = self.enabled

        reengage = self.reengage

        stop_after_days = self.stop_after_days

        stop_broadcasts = self.stop_broadcasts

        sunset_days = self.sunset_days

        confirm_erase = self.confirm_erase

        enabled_at = self.enabled_at

        reengage_automation = self.reengage_automation


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "activeDays": active_days,
            "atRiskDays": at_risk_days,
            "countMachineOpens": count_machine_opens,
            "deleteAfter": delete_after,
            "deleteAfterMonths": delete_after_months,
            "enabled": enabled,
            "reengage": reengage,
            "stopAfterDays": stop_after_days,
            "stopBroadcasts": stop_broadcasts,
            "sunsetDays": sunset_days,
        })
        if confirm_erase is not UNSET:
            field_dict["confirmErase"] = confirm_erase
        if enabled_at is not UNSET:
            field_dict["enabledAt"] = enabled_at
        if reengage_automation is not UNSET:
            field_dict["reengageAutomation"] = reengage_automation

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active_days = d.pop("activeDays")

        at_risk_days = d.pop("atRiskDays")

        count_machine_opens = d.pop("countMachineOpens")

        delete_after = d.pop("deleteAfter")

        delete_after_months = d.pop("deleteAfterMonths")

        enabled = d.pop("enabled")

        reengage = d.pop("reengage")

        stop_after_days = d.pop("stopAfterDays")

        stop_broadcasts = d.pop("stopBroadcasts")

        sunset_days = d.pop("sunsetDays")

        confirm_erase = d.pop("confirmErase", UNSET)

        enabled_at = d.pop("enabledAt", UNSET)

        reengage_automation = d.pop("reengageAutomation", UNSET)

        sunset_policy = cls(
            active_days=active_days,
            at_risk_days=at_risk_days,
            count_machine_opens=count_machine_opens,
            delete_after=delete_after,
            delete_after_months=delete_after_months,
            enabled=enabled,
            reengage=reengage,
            stop_after_days=stop_after_days,
            stop_broadcasts=stop_broadcasts,
            sunset_days=sunset_days,
            confirm_erase=confirm_erase,
            enabled_at=enabled_at,
            reengage_automation=reengage_automation,
        )

        return sunset_policy

