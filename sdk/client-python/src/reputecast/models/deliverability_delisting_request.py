from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.deliverability_delisting_outcome import DeliverabilityDelistingOutcome





T = TypeVar("T", bound="DeliverabilityDelistingRequest")



@_attrs_define
class DeliverabilityDelistingRequest:
    """ 
        Attributes:
            asset (str):
            requests (list[DeliverabilityDelistingOutcome]):
     """

    asset: str
    requests: list[DeliverabilityDelistingOutcome]





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_delisting_outcome import DeliverabilityDelistingOutcome # noqa: PLC0415
        asset = self.asset

        requests = []
        for requests_item_data in self.requests:
            requests_item = requests_item_data.to_dict()
            requests.append(requests_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asset": asset,
            "requests": requests,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_delisting_outcome import DeliverabilityDelistingOutcome # noqa: PLC0415
        d = dict(src_dict)
        asset = d.pop("asset")

        requests = []
        _requests = d.pop("requests")
        for requests_item_data in (_requests):
            requests_item = DeliverabilityDelistingOutcome.from_dict(requests_item_data)



            requests.append(requests_item)


        deliverability_delisting_request = cls(
            asset=asset,
            requests=requests,
        )

        return deliverability_delisting_request

