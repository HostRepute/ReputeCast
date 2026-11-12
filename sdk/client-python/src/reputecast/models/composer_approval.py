from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_approval_status import ComposerApprovalStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ComposerApproval")



@_attrs_define
class ComposerApproval:
    """ 
        Attributes:
            approved_by (list[str]):
            id (str):
            requested_at (datetime.datetime):
            requested_by (str):
            status (ComposerApprovalStatus):
            version (int):
            decided_at (datetime.datetime | Unset):
     """

    approved_by: list[str]
    id: str
    requested_at: datetime.datetime
    requested_by: str
    status: ComposerApprovalStatus
    version: int
    decided_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        approved_by = self.approved_by



        id = self.id

        requested_at = self.requested_at.isoformat()

        requested_by = self.requested_by

        status = self.status.value

        version = self.version

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvedBy": approved_by,
            "id": id,
            "requestedAt": requested_at,
            "requestedBy": requested_by,
            "status": status,
            "version": version,
        })
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approved_by = cast(list[str], d.pop("approvedBy"))


        id = d.pop("id")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        requested_by = d.pop("requestedBy")

        status = ComposerApprovalStatus(d.pop("status"))




        version = d.pop("version")

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        composer_approval = cls(
            approved_by=approved_by,
            id=id,
            requested_at=requested_at,
            requested_by=requested_by,
            status=status,
            version=version,
            decided_at=decided_at,
        )

        return composer_approval

