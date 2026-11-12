from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="SignupFormConsentEvidence")



@_attrs_define
class SignupFormConsentEvidence:
    """ 
        Attributes:
            at (datetime.datetime):
            contact_id (str):
            name (str):
            version (str):  Example: v1.
     """

    at: datetime.datetime
    contact_id: str
    name: str
    version: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        contact_id = self.contact_id

        name = self.name

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "contactId": contact_id,
            "name": name,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        contact_id = d.pop("contactId")

        name = d.pop("name")

        version = d.pop("version")

        signup_form_consent_evidence = cls(
            at=at,
            contact_id=contact_id,
            name=name,
            version=version,
        )

        return signup_form_consent_evidence

