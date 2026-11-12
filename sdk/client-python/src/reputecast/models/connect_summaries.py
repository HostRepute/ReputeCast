from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ConnectSummaries")



@_attrs_define
class ConnectSummaries:
    """ 
        Attributes:
            connection (str):
            domain (str):
            domain_short (str):
            provider (str):
            streams (str):
            test (str):
     """

    connection: str
    domain: str
    domain_short: str
    provider: str
    streams: str
    test: str





    def to_dict(self) -> dict[str, Any]:
        connection = self.connection

        domain = self.domain

        domain_short = self.domain_short

        provider = self.provider

        streams = self.streams

        test = self.test


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connection": connection,
            "domain": domain,
            "domainShort": domain_short,
            "provider": provider,
            "streams": streams,
            "test": test,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        connection = d.pop("connection")

        domain = d.pop("domain")

        domain_short = d.pop("domainShort")

        provider = d.pop("provider")

        streams = d.pop("streams")

        test = d.pop("test")

        connect_summaries = cls(
            connection=connection,
            domain=domain,
            domain_short=domain_short,
            provider=provider,
            streams=streams,
            test=test,
        )

        return connect_summaries

