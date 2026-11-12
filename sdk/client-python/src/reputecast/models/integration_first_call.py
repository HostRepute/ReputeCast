from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationFirstCall")



@_attrs_define
class IntegrationFirstCall:
    """ 
        Attributes:
            at (datetime.datetime):
            operation (str):
            status (int):
            client (str | Unset):
            ip (str | Unset):
     """

    at: datetime.datetime
    operation: str
    status: int
    client: str | Unset = UNSET
    ip: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        operation = self.operation

        status = self.status

        client = self.client

        ip = self.ip


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "operation": operation,
            "status": status,
        })
        if client is not UNSET:
            field_dict["client"] = client
        if ip is not UNSET:
            field_dict["ip"] = ip

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        operation = d.pop("operation")

        status = d.pop("status")

        client = d.pop("client", UNSET)

        ip = d.pop("ip", UNSET)

        integration_first_call = cls(
            at=at,
            operation=operation,
            status=status,
            client=client,
            ip=ip,
        )

        return integration_first_call

