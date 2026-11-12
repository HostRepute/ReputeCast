from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_events_input_events import ConnectEventsInputEvents






T = TypeVar("T", bound="ConnectEventsInput")



@_attrs_define
class ConnectEventsInput:
    """ 
        Attributes:
            events (ConnectEventsInputEvents):
     """

    events: ConnectEventsInputEvents





    def to_dict(self) -> dict[str, Any]:
        events = self.events.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "events": events,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        events = ConnectEventsInputEvents(d.pop("events"))




        connect_events_input = cls(
            events=events,
        )

        return connect_events_input

