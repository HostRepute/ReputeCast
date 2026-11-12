from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppConsentEvidence")



@_attrs_define
class WhatsAppConsentEvidence:
    """ 
        Attributes:
            at (datetime.datetime):
            consent_text (str):
            source (str): Where it was given: a form, a call, an event
            consent_version (str | Unset):
            ip (str | Unset):
            page (str | Unset):
     """

    at: datetime.datetime
    consent_text: str
    source: str
    consent_version: str | Unset = UNSET
    ip: str | Unset = UNSET
    page: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        consent_text = self.consent_text

        source = self.source

        consent_version = self.consent_version

        ip = self.ip

        page = self.page


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "consentText": consent_text,
            "source": source,
        })
        if consent_version is not UNSET:
            field_dict["consentVersion"] = consent_version
        if ip is not UNSET:
            field_dict["ip"] = ip
        if page is not UNSET:
            field_dict["page"] = page

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        consent_text = d.pop("consentText")

        source = d.pop("source")

        consent_version = d.pop("consentVersion", UNSET)

        ip = d.pop("ip", UNSET)

        page = d.pop("page", UNSET)

        whats_app_consent_evidence = cls(
            at=at,
            consent_text=consent_text,
            source=source,
            consent_version=consent_version,
            ip=ip,
            page=page,
        )

        return whats_app_consent_evidence

