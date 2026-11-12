from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralMilestoneEmail")



@_attrs_define
class ReferralMilestoneEmail:
    """ 
        Attributes:
            email_id (str):
            on (bool):
            subject (str):
            versions (int):
            saved_at (datetime.datetime | Unset):
     """

    email_id: str
    on: bool
    subject: str
    versions: int
    saved_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email_id = self.email_id

        on = self.on

        subject = self.subject

        versions = self.versions

        saved_at: str | Unset = UNSET
        if not isinstance(self.saved_at, Unset):
            saved_at = self.saved_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "emailId": email_id,
            "on": on,
            "subject": subject,
            "versions": versions,
        })
        if saved_at is not UNSET:
            field_dict["savedAt"] = saved_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email_id = d.pop("emailId")

        on = d.pop("on")

        subject = d.pop("subject")

        versions = d.pop("versions")

        _saved_at = d.pop("savedAt", UNSET)
        saved_at: datetime.datetime | Unset
        if isinstance(_saved_at,  Unset):
            saved_at = UNSET
        else:
            saved_at = datetime.datetime.fromisoformat(_saved_at)




        referral_milestone_email = cls(
            email_id=email_id,
            on=on,
            subject=subject,
            versions=versions,
            saved_at=saved_at,
        )

        return referral_milestone_email

