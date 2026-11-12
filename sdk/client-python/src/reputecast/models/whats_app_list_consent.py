from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppListConsent")



@_attrs_define
class WhatsAppListConsent:
    """ 
        Attributes:
            consent (int):
            list_id (str):
            members (int):
            name (str):
     """

    consent: int
    list_id: str
    members: int
    name: str





    def to_dict(self) -> dict[str, Any]:
        consent = self.consent

        list_id = self.list_id

        members = self.members

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "consent": consent,
            "listId": list_id,
            "members": members,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        consent = d.pop("consent")

        list_id = d.pop("listId")

        members = d.pop("members")

        name = d.pop("name")

        whats_app_list_consent = cls(
            consent=consent,
            list_id=list_id,
            members=members,
            name=name,
        )

        return whats_app_list_consent

