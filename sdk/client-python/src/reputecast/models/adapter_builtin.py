from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AdapterBuiltin")



@_attrs_define
class AdapterBuiltin:
    """ 
        Attributes:
            adapter (str):
            connected (int):
            domain_setup (str):
            events (str):
            maintainer (str):
            name (str):
            suppression_sync (str):
            version (str):
     """

    adapter: str
    connected: int
    domain_setup: str
    events: str
    maintainer: str
    name: str
    suppression_sync: str
    version: str





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        connected = self.connected

        domain_setup = self.domain_setup

        events = self.events

        maintainer = self.maintainer

        name = self.name

        suppression_sync = self.suppression_sync

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "connected": connected,
            "domainSetup": domain_setup,
            "events": events,
            "maintainer": maintainer,
            "name": name,
            "suppressionSync": suppression_sync,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        connected = d.pop("connected")

        domain_setup = d.pop("domainSetup")

        events = d.pop("events")

        maintainer = d.pop("maintainer")

        name = d.pop("name")

        suppression_sync = d.pop("suppressionSync")

        version = d.pop("version")

        adapter_builtin = cls(
            adapter=adapter,
            connected=connected,
            domain_setup=domain_setup,
            events=events,
            maintainer=maintainer,
            name=name,
            suppression_sync=suppression_sync,
            version=version,
        )

        return adapter_builtin

