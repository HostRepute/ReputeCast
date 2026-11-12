from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.inbound_mailbox_view_protocol import InboundMailboxViewProtocol
from ..models.inbound_mailbox_view_security import InboundMailboxViewSecurity
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="InboundMailboxView")



@_attrs_define
class InboundMailboxView:
    """ 
        Attributes:
            address (str):
            host (str):
            password_set (bool):
            port (int):
            protocol (InboundMailboxViewProtocol):
            security (InboundMailboxViewSecurity):
            username (str):
            last_polled_at (datetime.datetime | Unset):
     """

    address: str
    host: str
    password_set: bool
    port: int
    protocol: InboundMailboxViewProtocol
    security: InboundMailboxViewSecurity
    username: str
    last_polled_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        address = self.address

        host = self.host

        password_set = self.password_set

        port = self.port

        protocol = self.protocol.value

        security = self.security.value

        username = self.username

        last_polled_at: str | Unset = UNSET
        if not isinstance(self.last_polled_at, Unset):
            last_polled_at = self.last_polled_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "host": host,
            "passwordSet": password_set,
            "port": port,
            "protocol": protocol,
            "security": security,
            "username": username,
        })
        if last_polled_at is not UNSET:
            field_dict["lastPolledAt"] = last_polled_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        address = d.pop("address")

        host = d.pop("host")

        password_set = d.pop("passwordSet")

        port = d.pop("port")

        protocol = InboundMailboxViewProtocol(d.pop("protocol"))




        security = InboundMailboxViewSecurity(d.pop("security"))




        username = d.pop("username")

        _last_polled_at = d.pop("lastPolledAt", UNSET)
        last_polled_at: datetime.datetime | Unset
        if isinstance(_last_polled_at,  Unset):
            last_polled_at = UNSET
        else:
            last_polled_at = datetime.datetime.fromisoformat(_last_polled_at)




        inbound_mailbox_view = cls(
            address=address,
            host=host,
            password_set=password_set,
            port=port,
            protocol=protocol,
            security=security,
            username=username,
            last_polled_at=last_polled_at,
        )

        return inbound_mailbox_view

