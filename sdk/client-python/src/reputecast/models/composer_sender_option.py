from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ComposerSenderOption")



@_attrs_define
class ComposerSenderOption:
    """ 
        Attributes:
            domain (str):
            email (str):
            id (str):
            name (str):
            verified (bool):
     """

    domain: str
    email: str
    id: str
    name: str
    verified: bool





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        email = self.email

        id = self.id

        name = self.name

        verified = self.verified


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "email": email,
            "id": id,
            "name": name,
            "verified": verified,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        email = d.pop("email")

        id = d.pop("id")

        name = d.pop("name")

        verified = d.pop("verified")

        composer_sender_option = cls(
            domain=domain,
            email=email,
            id=id,
            name=name,
            verified=verified,
        )

        return composer_sender_option

