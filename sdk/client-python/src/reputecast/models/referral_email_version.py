from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReferralEmailVersion")



@_attrs_define
class ReferralEmailVersion:
    """ 
        Attributes:
            saved_at (datetime.datetime):
            saved_by (str):
            subject (str):
            version (int):
     """

    saved_at: datetime.datetime
    saved_by: str
    subject: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        saved_at = self.saved_at.isoformat()

        saved_by = self.saved_by

        subject = self.subject

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "savedAt": saved_at,
            "savedBy": saved_by,
            "subject": subject,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        saved_by = d.pop("savedBy")

        subject = d.pop("subject")

        version = d.pop("version")

        referral_email_version = cls(
            saved_at=saved_at,
            saved_by=saved_by,
            subject=subject,
            version=version,
        )

        return referral_email_version

