from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_send_settings_mode import PushSendSettingsMode
from ..models.push_send_settings_outside import PushSendSettingsOutside
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushSendSettings")



@_attrs_define
class PushSendSettings:
    """ 
        Attributes:
            mode (PushSendSettingsMode):
            outside (PushSendSettingsOutside):
            send_at (datetime.datetime | Unset):
     """

    mode: PushSendSettingsMode
    outside: PushSendSettingsOutside
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        outside = self.outside.value

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "outside": outside,
        })
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = PushSendSettingsMode(d.pop("mode"))




        outside = PushSendSettingsOutside(d.pop("outside"))




        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        push_send_settings = cls(
            mode=mode,
            outside=outside,
            send_at=send_at,
        )

        return push_send_settings

