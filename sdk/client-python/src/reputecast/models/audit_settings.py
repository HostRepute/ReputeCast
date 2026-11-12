from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.audit_stream_status import AuditStreamStatus





T = TypeVar("T", bound="AuditSettings")



@_attrs_define
class AuditSettings:
    """ 
        Attributes:
            can_manage (bool): Changing settings needs workspace:manage
            retention_days (int):
            stream (AuditStreamStatus):
            total (int):
     """

    can_manage: bool
    retention_days: int
    stream: AuditStreamStatus
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.audit_stream_status import AuditStreamStatus # noqa: PLC0415
        can_manage = self.can_manage

        retention_days = self.retention_days

        stream = self.stream.to_dict()

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "retentionDays": retention_days,
            "stream": stream,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audit_stream_status import AuditStreamStatus # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        retention_days = d.pop("retentionDays")

        stream = AuditStreamStatus.from_dict(d.pop("stream"))




        total = d.pop("total")

        audit_settings = cls(
            can_manage=can_manage,
            retention_days=retention_days,
            stream=stream,
            total=total,
        )

        return audit_settings

