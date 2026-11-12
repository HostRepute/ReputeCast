from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalSenderOption")



@_attrs_define
class TransactionalSenderOption:
    """ 
        Attributes:
            domain (str):
            email (str):
            id (str):
            name (str):
            reply_to (str):
            verified (bool):
     """

    domain: str
    email: str
    id: str
    name: str
    reply_to: str
    verified: bool





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        email = self.email

        id = self.id

        name = self.name

        reply_to = self.reply_to

        verified = self.verified


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "email": email,
            "id": id,
            "name": name,
            "replyTo": reply_to,
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

        reply_to = d.pop("replyTo")

        verified = d.pop("verified")

        transactional_sender_option = cls(
            domain=domain,
            email=email,
            id=id,
            name=name,
            reply_to=reply_to,
            verified=verified,
        )

        return transactional_sender_option

