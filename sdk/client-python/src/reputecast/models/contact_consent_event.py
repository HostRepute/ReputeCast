from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ContactConsentEvent")



@_attrs_define
class ContactConsentEvent:
    """ 
        Attributes:
            action (str):
            at (datetime.datetime):
            detail (str):
            note (str):
            source (str):
            source_label (str):
            actor (str | Unset):
            consent_text (str | Unset):
            consent_version (str | Unset):
            ip (str | Unset):
            language (str | Unset):
     """

    action: str
    at: datetime.datetime
    detail: str
    note: str
    source: str
    source_label: str
    actor: str | Unset = UNSET
    consent_text: str | Unset = UNSET
    consent_version: str | Unset = UNSET
    ip: str | Unset = UNSET
    language: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action

        at = self.at.isoformat()

        detail = self.detail

        note = self.note

        source = self.source

        source_label = self.source_label

        actor = self.actor

        consent_text = self.consent_text

        consent_version = self.consent_version

        ip = self.ip

        language = self.language


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "at": at,
            "detail": detail,
            "note": note,
            "source": source,
            "sourceLabel": source_label,
        })
        if actor is not UNSET:
            field_dict["actor"] = actor
        if consent_text is not UNSET:
            field_dict["consentText"] = consent_text
        if consent_version is not UNSET:
            field_dict["consentVersion"] = consent_version
        if ip is not UNSET:
            field_dict["ip"] = ip
        if language is not UNSET:
            field_dict["language"] = language

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = d.pop("action")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        detail = d.pop("detail")

        note = d.pop("note")

        source = d.pop("source")

        source_label = d.pop("sourceLabel")

        actor = d.pop("actor", UNSET)

        consent_text = d.pop("consentText", UNSET)

        consent_version = d.pop("consentVersion", UNSET)

        ip = d.pop("ip", UNSET)

        language = d.pop("language", UNSET)

        contact_consent_event = cls(
            action=action,
            at=at,
            detail=detail,
            note=note,
            source=source,
            source_label=source_label,
            actor=actor,
            consent_text=consent_text,
            consent_version=consent_version,
            ip=ip,
            language=language,
        )

        return contact_consent_event

