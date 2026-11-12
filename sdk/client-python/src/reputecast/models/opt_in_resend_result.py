from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="OptInResendResult")



@_attrs_define
class OptInResendResult:
    """ 
        Attributes:
            sent (bool):
            confirm_url (str | Unset):
     """

    sent: bool
    confirm_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        sent = self.sent

        confirm_url = self.confirm_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "sent": sent,
        })
        if confirm_url is not UNSET:
            field_dict["confirmUrl"] = confirm_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        sent = d.pop("sent")

        confirm_url = d.pop("confirmUrl", UNSET)

        opt_in_resend_result = cls(
            sent=sent,
            confirm_url=confirm_url,
        )

        return opt_in_resend_result

