from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_smtp_input_tls import ConnectSMTPInputTls
from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectSMTPInput")



@_attrs_define
class ConnectSMTPInput:
    """ 
        Attributes:
            host (str):
            port (int):
            tls (ConnectSMTPInputTls):
            daily_cap (str | Unset): Messages per day, e.g. 20,000
            max_rate (str | Unset): Messages per second the server allows, e.g. 5 per second
            password (str | Unset):
            username (str | Unset):
     """

    host: str
    port: int
    tls: ConnectSMTPInputTls
    daily_cap: str | Unset = UNSET
    max_rate: str | Unset = UNSET
    password: str | Unset = UNSET
    username: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        port = self.port

        tls = self.tls.value

        daily_cap = self.daily_cap

        max_rate = self.max_rate

        password = self.password

        username = self.username


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "port": port,
            "tls": tls,
        })
        if daily_cap is not UNSET:
            field_dict["dailyCap"] = daily_cap
        if max_rate is not UNSET:
            field_dict["maxRate"] = max_rate
        if password is not UNSET:
            field_dict["password"] = password
        if username is not UNSET:
            field_dict["username"] = username

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        port = d.pop("port")

        tls = ConnectSMTPInputTls(d.pop("tls"))




        daily_cap = d.pop("dailyCap", UNSET)

        max_rate = d.pop("maxRate", UNSET)

        password = d.pop("password", UNSET)

        username = d.pop("username", UNSET)

        connect_smtp_input = cls(
            host=host,
            port=port,
            tls=tls,
            daily_cap=daily_cap,
            max_rate=max_rate,
            password=password,
            username=username,
        )

        return connect_smtp_input

