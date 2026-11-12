from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.workspace_retention_event_detail_days import WorkspaceRetentionEventDetailDays
from ..models.workspace_retention_unconfirmed_erase_days import WorkspaceRetentionUnconfirmedEraseDays






T = TypeVar("T", bound="WorkspaceRetention")



@_attrs_define
class WorkspaceRetention:
    """ 
        Attributes:
            event_detail_days (WorkspaceRetentionEventDetailDays):
            unconfirmed_erase_days (WorkspaceRetentionUnconfirmedEraseDays):
     """

    event_detail_days: WorkspaceRetentionEventDetailDays
    unconfirmed_erase_days: WorkspaceRetentionUnconfirmedEraseDays





    def to_dict(self) -> dict[str, Any]:
        event_detail_days = self.event_detail_days.value

        unconfirmed_erase_days = self.unconfirmed_erase_days.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "eventDetailDays": event_detail_days,
            "unconfirmedEraseDays": unconfirmed_erase_days,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        event_detail_days = WorkspaceRetentionEventDetailDays(d.pop("eventDetailDays"))




        unconfirmed_erase_days = WorkspaceRetentionUnconfirmedEraseDays(d.pop("unconfirmedEraseDays"))




        workspace_retention = cls(
            event_detail_days=event_detail_days,
            unconfirmed_erase_days=unconfirmed_erase_days,
        )

        return workspace_retention

