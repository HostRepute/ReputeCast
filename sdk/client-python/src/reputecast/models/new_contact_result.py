from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="NewContactResult")



@_attrs_define
class NewContactResult:
    """ 
        Attributes:
            confirmation_sent (bool):
            id (str):
            confirm_url (str | Unset): Shown when no System provider could send the confirmation
     """

    confirmation_sent: bool
    id: str
    confirm_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        confirmation_sent = self.confirmation_sent

        id = self.id

        confirm_url = self.confirm_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmationSent": confirmation_sent,
            "id": id,
        })
        if confirm_url is not UNSET:
            field_dict["confirmUrl"] = confirm_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confirmation_sent = d.pop("confirmationSent")

        id = d.pop("id")

        confirm_url = d.pop("confirmUrl", UNSET)

        new_contact_result = cls(
            confirmation_sent=confirmation_sent,
            id=id,
            confirm_url=confirm_url,
        )

        return new_contact_result

