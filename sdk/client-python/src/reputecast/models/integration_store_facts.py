from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_store_facts_state import IntegrationStoreFactsState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationStoreFacts")



@_attrs_define
class IntegrationStoreFacts:
    """ 
        Attributes:
            name (str):
            orders (int):
            state (IntegrationStoreFactsState):
            stores (int):
            last_event_at (datetime.datetime | Unset):
     """

    name: str
    orders: int
    state: IntegrationStoreFactsState
    stores: int
    last_event_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        orders = self.orders

        state = self.state.value

        stores = self.stores

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "orders": orders,
            "state": state,
            "stores": stores,
        })
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        orders = d.pop("orders")

        state = IntegrationStoreFactsState(d.pop("state"))




        stores = d.pop("stores")

        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        integration_store_facts = cls(
            name=name,
            orders=orders,
            state=state,
            stores=stores,
            last_event_at=last_event_at,
        )

        return integration_store_facts

