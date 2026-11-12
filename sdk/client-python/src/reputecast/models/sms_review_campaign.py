from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_review_campaign_status import SmsReviewCampaignStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsReviewCampaign")



@_attrs_define
class SmsReviewCampaign:
    """ 
        Attributes:
            id (str):
            name (str):
            status (SmsReviewCampaignStatus):
            text (str):
            updated_by (str):
            version (int):
            autosaved_at (datetime.datetime | Unset):
     """

    id: str
    name: str
    status: SmsReviewCampaignStatus
    text: str
    updated_by: str
    version: int
    autosaved_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        status = self.status.value

        text = self.text

        updated_by = self.updated_by

        version = self.version

        autosaved_at: str | Unset = UNSET
        if not isinstance(self.autosaved_at, Unset):
            autosaved_at = self.autosaved_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "status": status,
            "text": text,
            "updatedBy": updated_by,
            "version": version,
        })
        if autosaved_at is not UNSET:
            field_dict["autosavedAt"] = autosaved_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        status = SmsReviewCampaignStatus(d.pop("status"))




        text = d.pop("text")

        updated_by = d.pop("updatedBy")

        version = d.pop("version")

        _autosaved_at = d.pop("autosavedAt", UNSET)
        autosaved_at: datetime.datetime | Unset
        if isinstance(_autosaved_at,  Unset):
            autosaved_at = UNSET
        else:
            autosaved_at = datetime.datetime.fromisoformat(_autosaved_at)




        sms_review_campaign = cls(
            id=id,
            name=name,
            status=status,
            text=text,
            updated_by=updated_by,
            version=version,
            autosaved_at=autosaved_at,
        )

        return sms_review_campaign

