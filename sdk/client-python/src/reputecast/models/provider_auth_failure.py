from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.provider_down_state import ProviderDownState
  from ..models.provider_outage_stream import ProviderOutageStream





T = TypeVar("T", bound="ProviderAuthFailure")



@_attrs_define
class ProviderAuthFailure:
    """ 
        Attributes:
            down (ProviderDownState):
            provider (str):
            provider_id (str):
            rerouted (int):
            short (str):
            streams (list[ProviderOutageStream]):
     """

    down: ProviderDownState
    provider: str
    provider_id: str
    rerouted: int
    short: str
    streams: list[ProviderOutageStream]





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_down_state import ProviderDownState # noqa: PLC0415
        from ..models.provider_outage_stream import ProviderOutageStream # noqa: PLC0415
        down = self.down.to_dict()

        provider = self.provider

        provider_id = self.provider_id

        rerouted = self.rerouted

        short = self.short

        streams = []
        for streams_item_data in self.streams:
            streams_item = streams_item_data.to_dict()
            streams.append(streams_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "down": down,
            "provider": provider,
            "providerId": provider_id,
            "rerouted": rerouted,
            "short": short,
            "streams": streams,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_down_state import ProviderDownState # noqa: PLC0415
        from ..models.provider_outage_stream import ProviderOutageStream # noqa: PLC0415
        d = dict(src_dict)
        down = ProviderDownState.from_dict(d.pop("down"))




        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        rerouted = d.pop("rerouted")

        short = d.pop("short")

        streams = []
        _streams = d.pop("streams")
        for streams_item_data in (_streams):
            streams_item = ProviderOutageStream.from_dict(streams_item_data)



            streams.append(streams_item)


        provider_auth_failure = cls(
            down=down,
            provider=provider,
            provider_id=provider_id,
            rerouted=rerouted,
            short=short,
            streams=streams,
        )

        return provider_auth_failure

