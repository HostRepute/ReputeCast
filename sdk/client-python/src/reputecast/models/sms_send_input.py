from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SmsSendInput")



@_attrs_define
class SmsSendInput:
    """ 
        Attributes:
            fingerprint (str):
            acknowledged (list[str] | Unset):
            confirm (str | Unset):
     """

    fingerprint: str
    acknowledged: list[str] | Unset = UNSET
    confirm: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        fingerprint = self.fingerprint

        acknowledged: list[str] | Unset = UNSET
        if not isinstance(self.acknowledged, Unset):
            acknowledged = self.acknowledged



        confirm = self.confirm


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fingerprint": fingerprint,
        })
        if acknowledged is not UNSET:
            field_dict["acknowledged"] = acknowledged
        if confirm is not UNSET:
            field_dict["confirm"] = confirm

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fingerprint = d.pop("fingerprint")

        acknowledged = cast(list[str], d.pop("acknowledged", UNSET))


        confirm = d.pop("confirm", UNSET)

        sms_send_input = cls(
            fingerprint=fingerprint,
            acknowledged=acknowledged,
            confirm=confirm,
        )

        return sms_send_input

