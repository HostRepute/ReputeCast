from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_reward_item_status import ReferralRewardItemStatus
from ..models.referral_reward_item_type import ReferralRewardItemType
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralRewardItem")



@_attrs_define
class ReferralRewardItem:
    """ 
        Attributes:
            error (str):
            file_name (str):
            file_size (int):
            file_type (str):
            has_claim (bool):
            id (str):
            milestone (int):
            name (str):
            note (str):
            reached_at (datetime.datetime):
            status (ReferralRewardItemStatus):
            type_ (ReferralRewardItemType):
            code (str | Unset):
            expires_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
            used_at (datetime.datetime | Unset):
     """

    error: str
    file_name: str
    file_size: int
    file_type: str
    has_claim: bool
    id: str
    milestone: int
    name: str
    note: str
    reached_at: datetime.datetime
    status: ReferralRewardItemStatus
    type_: ReferralRewardItemType
    code: str | Unset = UNSET
    expires_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    used_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        error = self.error

        file_name = self.file_name

        file_size = self.file_size

        file_type = self.file_type

        has_claim = self.has_claim

        id = self.id

        milestone = self.milestone

        name = self.name

        note = self.note

        reached_at = self.reached_at.isoformat()

        status = self.status.value

        type_ = self.type_.value

        code = self.code

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        used_at: str | Unset = UNSET
        if not isinstance(self.used_at, Unset):
            used_at = self.used_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "error": error,
            "fileName": file_name,
            "fileSize": file_size,
            "fileType": file_type,
            "hasClaim": has_claim,
            "id": id,
            "milestone": milestone,
            "name": name,
            "note": note,
            "reachedAt": reached_at,
            "status": status,
            "type": type_,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if used_at is not UNSET:
            field_dict["usedAt"] = used_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        error = d.pop("error")

        file_name = d.pop("fileName")

        file_size = d.pop("fileSize")

        file_type = d.pop("fileType")

        has_claim = d.pop("hasClaim")

        id = d.pop("id")

        milestone = d.pop("milestone")

        name = d.pop("name")

        note = d.pop("note")

        reached_at = datetime.datetime.fromisoformat(d.pop("reachedAt"))




        status = ReferralRewardItemStatus(d.pop("status"))




        type_ = ReferralRewardItemType(d.pop("type"))




        code = d.pop("code", UNSET)

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        _used_at = d.pop("usedAt", UNSET)
        used_at: datetime.datetime | Unset
        if isinstance(_used_at,  Unset):
            used_at = UNSET
        else:
            used_at = datetime.datetime.fromisoformat(_used_at)




        referral_reward_item = cls(
            error=error,
            file_name=file_name,
            file_size=file_size,
            file_type=file_type,
            has_claim=has_claim,
            id=id,
            milestone=milestone,
            name=name,
            note=note,
            reached_at=reached_at,
            status=status,
            type_=type_,
            code=code,
            expires_at=expires_at,
            sent_at=sent_at,
            used_at=used_at,
        )

        return referral_reward_item

