from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_version_item_status import AutomationVersionItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AutomationVersionItem")



@_attrs_define
class AutomationVersionItem:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by (str):
            in_flow (int):
            status (AutomationVersionItemStatus): retiring: replaced, but contacts are still finishing on it
            summary (str):
            version (int):
            from_creation (bool | Unset):
            published_at (datetime.datetime | Unset):
            published_by (str | Unset):
            retired_at (datetime.datetime | Unset):
     """

    created_at: datetime.datetime
    created_by: str
    in_flow: int
    status: AutomationVersionItemStatus
    summary: str
    version: int
    from_creation: bool | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    published_by: str | Unset = UNSET
    retired_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        created_by = self.created_by

        in_flow = self.in_flow

        status = self.status.value

        summary = self.summary

        version = self.version

        from_creation = self.from_creation

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by = self.published_by

        retired_at: str | Unset = UNSET
        if not isinstance(self.retired_at, Unset):
            retired_at = self.retired_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdBy": created_by,
            "inFlow": in_flow,
            "status": status,
            "summary": summary,
            "version": version,
        })
        if from_creation is not UNSET:
            field_dict["fromCreation"] = from_creation
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by
        if retired_at is not UNSET:
            field_dict["retiredAt"] = retired_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        in_flow = d.pop("inFlow")

        status = AutomationVersionItemStatus(d.pop("status"))




        summary = d.pop("summary")

        version = d.pop("version")

        from_creation = d.pop("fromCreation", UNSET)

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        published_by = d.pop("publishedBy", UNSET)

        _retired_at = d.pop("retiredAt", UNSET)
        retired_at: datetime.datetime | Unset
        if isinstance(_retired_at,  Unset):
            retired_at = UNSET
        else:
            retired_at = datetime.datetime.fromisoformat(_retired_at)




        automation_version_item = cls(
            created_at=created_at,
            created_by=created_by,
            in_flow=in_flow,
            status=status,
            summary=summary,
            version=version,
            from_creation=from_creation,
            published_at=published_at,
            published_by=published_by,
            retired_at=retired_at,
        )

        return automation_version_item

