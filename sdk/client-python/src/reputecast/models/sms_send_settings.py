from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_send_settings_mode import SmsSendSettingsMode
from ..models.sms_send_settings_pending_policy import SmsSendSettingsPendingPolicy
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_thresholds import SmsThresholds





T = TypeVar("T", bound="SmsSendSettings")



@_attrs_define
class SmsSendSettings:
    """ 
        Attributes:
            local_date (str | Unset):
            local_time (str | Unset):
            mode (SmsSendSettingsMode | Unset):
            pending_policy (SmsSendSettingsPendingPolicy | Unset):
            send_at (datetime.datetime | Unset):
            thresholds (SmsThresholds | Unset):
     """

    local_date: str | Unset = UNSET
    local_time: str | Unset = UNSET
    mode: SmsSendSettingsMode | Unset = UNSET
    pending_policy: SmsSendSettingsPendingPolicy | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    thresholds: SmsThresholds | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        local_date = self.local_date

        local_time = self.local_time

        mode: str | Unset = UNSET
        if not isinstance(self.mode, Unset):
            mode = self.mode.value


        pending_policy: str | Unset = UNSET
        if not isinstance(self.pending_policy, Unset):
            pending_policy = self.pending_policy.value


        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        thresholds: dict[str, Any] | Unset = UNSET
        if not isinstance(self.thresholds, Unset):
            thresholds = self.thresholds.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if local_date is not UNSET:
            field_dict["localDate"] = local_date
        if local_time is not UNSET:
            field_dict["localTime"] = local_time
        if mode is not UNSET:
            field_dict["mode"] = mode
        if pending_policy is not UNSET:
            field_dict["pendingPolicy"] = pending_policy
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at
        if thresholds is not UNSET:
            field_dict["thresholds"] = thresholds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        d = dict(src_dict)
        local_date = d.pop("localDate", UNSET)

        local_time = d.pop("localTime", UNSET)

        _mode = d.pop("mode", UNSET)
        mode: SmsSendSettingsMode | Unset
        if isinstance(_mode,  Unset):
            mode = UNSET
        else:
            mode = SmsSendSettingsMode(_mode)




        _pending_policy = d.pop("pendingPolicy", UNSET)
        pending_policy: SmsSendSettingsPendingPolicy | Unset
        if isinstance(_pending_policy,  Unset):
            pending_policy = UNSET
        else:
            pending_policy = SmsSendSettingsPendingPolicy(_pending_policy)




        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        _thresholds = d.pop("thresholds", UNSET)
        thresholds: SmsThresholds | Unset
        if isinstance(_thresholds,  Unset):
            thresholds = UNSET
        else:
            thresholds = SmsThresholds.from_dict(_thresholds)




        sms_send_settings = cls(
            local_date=local_date,
            local_time=local_time,
            mode=mode,
            pending_policy=pending_policy,
            send_at=send_at,
            thresholds=thresholds,
        )

        return sms_send_settings

