from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ApprovalCampaignView")



@_attrs_define
class ApprovalCampaignView:
    """ 
        Attributes:
            current_version (int):
            from_ (str):
            id (str):
            preheader (str):
            status (str):
            subject (str):
            version (int):
            edited_at (datetime.datetime | Unset):
            edited_by (str | Unset):
            scheduled_at (datetime.datetime | Unset):
     """

    current_version: int
    from_: str
    id: str
    preheader: str
    status: str
    subject: str
    version: int
    edited_at: datetime.datetime | Unset = UNSET
    edited_by: str | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        current_version = self.current_version

        from_ = self.from_

        id = self.id

        preheader = self.preheader

        status = self.status

        subject = self.subject

        version = self.version

        edited_at: str | Unset = UNSET
        if not isinstance(self.edited_at, Unset):
            edited_at = self.edited_at.isoformat()

        edited_by = self.edited_by

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currentVersion": current_version,
            "from": from_,
            "id": id,
            "preheader": preheader,
            "status": status,
            "subject": subject,
            "version": version,
        })
        if edited_at is not UNSET:
            field_dict["editedAt"] = edited_at
        if edited_by is not UNSET:
            field_dict["editedBy"] = edited_by
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        current_version = d.pop("currentVersion")

        from_ = d.pop("from")

        id = d.pop("id")

        preheader = d.pop("preheader")

        status = d.pop("status")

        subject = d.pop("subject")

        version = d.pop("version")

        _edited_at = d.pop("editedAt", UNSET)
        edited_at: datetime.datetime | Unset
        if isinstance(_edited_at,  Unset):
            edited_at = UNSET
        else:
            edited_at = datetime.datetime.fromisoformat(_edited_at)




        edited_by = d.pop("editedBy", UNSET)

        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        approval_campaign_view = cls(
            current_version=current_version,
            from_=from_,
            id=id,
            preheader=preheader,
            status=status,
            subject=subject,
            version=version,
            edited_at=edited_at,
            edited_by=edited_by,
            scheduled_at=scheduled_at,
        )

        return approval_campaign_view

