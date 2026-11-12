from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_webhook_test_status import ConnectWebhookTestStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectWebhookTest")



@_attrs_define
class ConnectWebhookTest:
    """ 
        Attributes:
            requested_at (datetime.datetime):
            status (ConnectWebhookTestStatus):
            until (datetime.datetime):
            arrived_at (datetime.datetime | Unset):
            error (str | Unset):
     """

    requested_at: datetime.datetime
    status: ConnectWebhookTestStatus
    until: datetime.datetime
    arrived_at: datetime.datetime | Unset = UNSET
    error: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        requested_at = self.requested_at.isoformat()

        status = self.status.value

        until = self.until.isoformat()

        arrived_at: str | Unset = UNSET
        if not isinstance(self.arrived_at, Unset):
            arrived_at = self.arrived_at.isoformat()

        error = self.error


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "requestedAt": requested_at,
            "status": status,
            "until": until,
        })
        if arrived_at is not UNSET:
            field_dict["arrivedAt"] = arrived_at
        if error is not UNSET:
            field_dict["error"] = error

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        status = ConnectWebhookTestStatus(d.pop("status"))




        until = datetime.datetime.fromisoformat(d.pop("until"))




        _arrived_at = d.pop("arrivedAt", UNSET)
        arrived_at: datetime.datetime | Unset
        if isinstance(_arrived_at,  Unset):
            arrived_at = UNSET
        else:
            arrived_at = datetime.datetime.fromisoformat(_arrived_at)




        error = d.pop("error", UNSET)

        connect_webhook_test = cls(
            requested_at=requested_at,
            status=status,
            until=until,
            arrived_at=arrived_at,
            error=error,
        )

        return connect_webhook_test

