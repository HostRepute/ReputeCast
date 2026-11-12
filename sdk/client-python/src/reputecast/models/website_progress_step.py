from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="WebsiteProgressStep")



@_attrs_define
class WebsiteProgressStep:
    """ 
        Attributes:
            done (int):
            total (int):
            names (list[str] | Unset):
     """

    done: int
    total: int
    names: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        done = self.done

        total = self.total

        names: list[str] | Unset = UNSET
        if not isinstance(self.names, Unset):
            names = self.names




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "done": done,
            "total": total,
        })
        if names is not UNSET:
            field_dict["names"] = names

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        done = d.pop("done")

        total = d.pop("total")

        names = cast(list[str], d.pop("names", UNSET))


        website_progress_step = cls(
            done=done,
            total=total,
            names=names,
        )

        return website_progress_step

