from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_consent_event_action import SmsConsentEventAction
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsConsentEvent")



@_attrs_define
class SmsConsentEvent:
    """ 
        Attributes:
            action (SmsConsentEventAction):
            at (datetime.datetime):
            id (str):
            number (str):
            source (str):
            actor (str | Unset):
            consent_text (str | Unset):
            consent_version (str | Unset):
            ip (str | Unset):
            keyword (str | Unset):
            list_ (str | Unset):
            note (str | Unset):
            page (str | Unset):
     """

    action: SmsConsentEventAction
    at: datetime.datetime
    id: str
    number: str
    source: str
    actor: str | Unset = UNSET
    consent_text: str | Unset = UNSET
    consent_version: str | Unset = UNSET
    ip: str | Unset = UNSET
    keyword: str | Unset = UNSET
    list_: str | Unset = UNSET
    note: str | Unset = UNSET
    page: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        at = self.at.isoformat()

        id = self.id

        number = self.number

        source = self.source

        actor = self.actor

        consent_text = self.consent_text

        consent_version = self.consent_version

        ip = self.ip

        keyword = self.keyword

        list_ = self.list_

        note = self.note

        page = self.page


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "at": at,
            "id": id,
            "number": number,
            "source": source,
        })
        if actor is not UNSET:
            field_dict["actor"] = actor
        if consent_text is not UNSET:
            field_dict["consentText"] = consent_text
        if consent_version is not UNSET:
            field_dict["consentVersion"] = consent_version
        if ip is not UNSET:
            field_dict["ip"] = ip
        if keyword is not UNSET:
            field_dict["keyword"] = keyword
        if list_ is not UNSET:
            field_dict["list"] = list_
        if note is not UNSET:
            field_dict["note"] = note
        if page is not UNSET:
            field_dict["page"] = page

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = SmsConsentEventAction(d.pop("action"))




        at = datetime.datetime.fromisoformat(d.pop("at"))




        id = d.pop("id")

        number = d.pop("number")

        source = d.pop("source")

        actor = d.pop("actor", UNSET)

        consent_text = d.pop("consentText", UNSET)

        consent_version = d.pop("consentVersion", UNSET)

        ip = d.pop("ip", UNSET)

        keyword = d.pop("keyword", UNSET)

        list_ = d.pop("list", UNSET)

        note = d.pop("note", UNSET)

        page = d.pop("page", UNSET)

        sms_consent_event = cls(
            action=action,
            at=at,
            id=id,
            number=number,
            source=source,
            actor=actor,
            consent_text=consent_text,
            consent_version=consent_version,
            ip=ip,
            keyword=keyword,
            list_=list_,
            note=note,
            page=page,
        )

        return sms_consent_event

