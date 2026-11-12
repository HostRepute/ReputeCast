from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.digest_person import DigestPerson





T = TypeVar("T", bound="DigestIssueApproval")



@_attrs_define
class DigestIssueApproval:
    """ 
        Attributes:
            id (str):
            requested_at (datetime.datetime):
            rules (list[str]):
            status (str):
            decided_at (datetime.datetime | Unset):
            decided_by (DigestPerson | Unset):
     """

    id: str
    requested_at: datetime.datetime
    rules: list[str]
    status: str
    decided_at: datetime.datetime | Unset = UNSET
    decided_by: DigestPerson | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        id = self.id

        requested_at = self.requested_at.isoformat()

        rules = self.rules



        status = self.status

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()

        decided_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.decided_by, Unset):
            decided_by = self.decided_by.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "requestedAt": requested_at,
            "rules": rules,
            "status": status,
        })
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at
        if decided_by is not UNSET:
            field_dict["decidedBy"] = decided_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        rules = cast(list[str], d.pop("rules"))


        status = d.pop("status")

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        _decided_by = d.pop("decidedBy", UNSET)
        decided_by: DigestPerson | Unset
        if isinstance(_decided_by,  Unset):
            decided_by = UNSET
        else:
            decided_by = DigestPerson.from_dict(_decided_by)




        digest_issue_approval = cls(
            id=id,
            requested_at=requested_at,
            rules=rules,
            status=status,
            decided_at=decided_at,
            decided_by=decided_by,
        )

        return digest_issue_approval

