from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="SmsCampaignVersion")



@_attrs_define
class SmsCampaignVersion:
    """ 
        Attributes:
            body (str):
            saved_at (datetime.datetime):
            saved_by (str):
            version (int):
     """

    body: str
    saved_at: datetime.datetime
    saved_by: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        body = self.body

        saved_at = self.saved_at.isoformat()

        saved_by = self.saved_by

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "savedAt": saved_at,
            "savedBy": saved_by,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        body = d.pop("body")

        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        saved_by = d.pop("savedBy")

        version = d.pop("version")

        sms_campaign_version = cls(
            body=body,
            saved_at=saved_at,
            saved_by=saved_by,
            version=version,
        )

        return sms_campaign_version

