from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sender_verification import SenderVerification





T = TypeVar("T", bound="SenderIdentity")



@_attrs_define
class SenderIdentity:
    """ 
        Attributes:
            default_for (list[str]):
            domain (str):
            domain_id (str):
            email (str):
            id (str):
            name (str):
            note (str):
            pending (bool):
            reply_to (str):
            smtp_relay (bool):
            transactional (bool):
            verification (list[SenderVerification]):
     """

    default_for: list[str]
    domain: str
    domain_id: str
    email: str
    id: str
    name: str
    note: str
    pending: bool
    reply_to: str
    smtp_relay: bool
    transactional: bool
    verification: list[SenderVerification]





    def to_dict(self) -> dict[str, Any]:
        from ..models.sender_verification import SenderVerification # noqa: PLC0415
        default_for = self.default_for



        domain = self.domain

        domain_id = self.domain_id

        email = self.email

        id = self.id

        name = self.name

        note = self.note

        pending = self.pending

        reply_to = self.reply_to

        smtp_relay = self.smtp_relay

        transactional = self.transactional

        verification = []
        for verification_item_data in self.verification:
            verification_item = verification_item_data.to_dict()
            verification.append(verification_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "defaultFor": default_for,
            "domain": domain,
            "domainId": domain_id,
            "email": email,
            "id": id,
            "name": name,
            "note": note,
            "pending": pending,
            "replyTo": reply_to,
            "smtpRelay": smtp_relay,
            "transactional": transactional,
            "verification": verification,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sender_verification import SenderVerification # noqa: PLC0415
        d = dict(src_dict)
        default_for = cast(list[str], d.pop("defaultFor"))


        domain = d.pop("domain")

        domain_id = d.pop("domainId")

        email = d.pop("email")

        id = d.pop("id")

        name = d.pop("name")

        note = d.pop("note")

        pending = d.pop("pending")

        reply_to = d.pop("replyTo")

        smtp_relay = d.pop("smtpRelay")

        transactional = d.pop("transactional")

        verification = []
        _verification = d.pop("verification")
        for verification_item_data in (_verification):
            verification_item = SenderVerification.from_dict(verification_item_data)



            verification.append(verification_item)


        sender_identity = cls(
            default_for=default_for,
            domain=domain,
            domain_id=domain_id,
            email=email,
            id=id,
            name=name,
            note=note,
            pending=pending,
            reply_to=reply_to,
            smtp_relay=smtp_relay,
            transactional=transactional,
            verification=verification,
        )

        return sender_identity

