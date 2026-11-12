from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_list_item_consent import ContactListItemConsent
from ..models.contact_list_item_score_band import ContactListItemScoreBand
from ..models.contact_list_item_tier import ContactListItemTier
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.contact_list_item_fields import ContactListItemFields
  from ..models.contact_paid import ContactPaid





T = TypeVar("T", bound="ContactListItem")



@_attrs_define
class ContactListItem:
    """ 
        Attributes:
            consent (ContactListItemConsent):
            email (str):
            id (str):
            joined_at (datetime.datetime):
            last_open_machine (bool): The last open was a machine open (Apple Mail Privacy Protection)
            lists (list[str]):
            mailbox_label (str): Outlook, or Other · Google Workspace
            mailbox_provider (str):
            name (str):
            source (str):
            source_label (str):
            suppressed (bool):
            tier (ContactListItemTier):
            fields (ContactListItemFields | Unset):
            last_open_at (datetime.datetime | Unset):
            paid (ContactPaid | Unset):
            phone (str | Unset):
            score (int | Unset):
            score_band (ContactListItemScoreBand | Unset):
     """

    consent: ContactListItemConsent
    email: str
    id: str
    joined_at: datetime.datetime
    last_open_machine: bool
    lists: list[str]
    mailbox_label: str
    mailbox_provider: str
    name: str
    source: str
    source_label: str
    suppressed: bool
    tier: ContactListItemTier
    fields: ContactListItemFields | Unset = UNSET
    last_open_at: datetime.datetime | Unset = UNSET
    paid: ContactPaid | Unset = UNSET
    phone: str | Unset = UNSET
    score: int | Unset = UNSET
    score_band: ContactListItemScoreBand | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_list_item_fields import ContactListItemFields # noqa: PLC0415
        from ..models.contact_paid import ContactPaid # noqa: PLC0415
        consent = self.consent.value

        email = self.email

        id = self.id

        joined_at = self.joined_at.isoformat()

        last_open_machine = self.last_open_machine

        lists = self.lists



        mailbox_label = self.mailbox_label

        mailbox_provider = self.mailbox_provider

        name = self.name

        source = self.source

        source_label = self.source_label

        suppressed = self.suppressed

        tier = self.tier.value

        fields: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fields, Unset):
            fields = self.fields.to_dict()

        last_open_at: str | Unset = UNSET
        if not isinstance(self.last_open_at, Unset):
            last_open_at = self.last_open_at.isoformat()

        paid: dict[str, Any] | Unset = UNSET
        if not isinstance(self.paid, Unset):
            paid = self.paid.to_dict()

        phone = self.phone

        score = self.score

        score_band: str | Unset = UNSET
        if not isinstance(self.score_band, Unset):
            score_band = self.score_band.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "consent": consent,
            "email": email,
            "id": id,
            "joinedAt": joined_at,
            "lastOpenMachine": last_open_machine,
            "lists": lists,
            "mailboxLabel": mailbox_label,
            "mailboxProvider": mailbox_provider,
            "name": name,
            "source": source,
            "sourceLabel": source_label,
            "suppressed": suppressed,
            "tier": tier,
        })
        if fields is not UNSET:
            field_dict["fields"] = fields
        if last_open_at is not UNSET:
            field_dict["lastOpenAt"] = last_open_at
        if paid is not UNSET:
            field_dict["paid"] = paid
        if phone is not UNSET:
            field_dict["phone"] = phone
        if score is not UNSET:
            field_dict["score"] = score
        if score_band is not UNSET:
            field_dict["scoreBand"] = score_band

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_list_item_fields import ContactListItemFields # noqa: PLC0415
        from ..models.contact_paid import ContactPaid # noqa: PLC0415
        d = dict(src_dict)
        consent = ContactListItemConsent(d.pop("consent"))




        email = d.pop("email")

        id = d.pop("id")

        joined_at = datetime.datetime.fromisoformat(d.pop("joinedAt"))




        last_open_machine = d.pop("lastOpenMachine")

        lists = cast(list[str], d.pop("lists"))


        mailbox_label = d.pop("mailboxLabel")

        mailbox_provider = d.pop("mailboxProvider")

        name = d.pop("name")

        source = d.pop("source")

        source_label = d.pop("sourceLabel")

        suppressed = d.pop("suppressed")

        tier = ContactListItemTier(d.pop("tier"))




        _fields = d.pop("fields", UNSET)
        fields: ContactListItemFields | Unset
        if isinstance(_fields,  Unset):
            fields = UNSET
        else:
            fields = ContactListItemFields.from_dict(_fields)




        _last_open_at = d.pop("lastOpenAt", UNSET)
        last_open_at: datetime.datetime | Unset
        if isinstance(_last_open_at,  Unset):
            last_open_at = UNSET
        else:
            last_open_at = datetime.datetime.fromisoformat(_last_open_at)




        _paid = d.pop("paid", UNSET)
        paid: ContactPaid | Unset
        if isinstance(_paid,  Unset):
            paid = UNSET
        else:
            paid = ContactPaid.from_dict(_paid)




        phone = d.pop("phone", UNSET)

        score = d.pop("score", UNSET)

        _score_band = d.pop("scoreBand", UNSET)
        score_band: ContactListItemScoreBand | Unset
        if isinstance(_score_band,  Unset):
            score_band = UNSET
        else:
            score_band = ContactListItemScoreBand(_score_band)




        contact_list_item = cls(
            consent=consent,
            email=email,
            id=id,
            joined_at=joined_at,
            last_open_machine=last_open_machine,
            lists=lists,
            mailbox_label=mailbox_label,
            mailbox_provider=mailbox_provider,
            name=name,
            source=source,
            source_label=source_label,
            suppressed=suppressed,
            tier=tier,
            fields=fields,
            last_open_at=last_open_at,
            paid=paid,
            phone=phone,
            score=score,
            score_band=score_band,
        )

        return contact_list_item

