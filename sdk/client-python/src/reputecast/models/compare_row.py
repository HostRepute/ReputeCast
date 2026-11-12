from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.compare_row_state import CompareRowState
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.block_change import BlockChange
  from ..models.compare_block import CompareBlock





T = TypeVar("T", bound="CompareRow")



@_attrs_define
class CompareRow:
    """ 
        Attributes:
            before (str):
            from_version (int):
            key (str):
            number (int):
            state (CompareRowState):
            changed (BlockChange | Unset):
            source (CompareBlock | Unset):
            version (CompareBlock | Unset):
     """

    before: str
    from_version: int
    key: str
    number: int
    state: CompareRowState
    changed: BlockChange | Unset = UNSET
    source: CompareBlock | Unset = UNSET
    version: CompareBlock | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.block_change import BlockChange # noqa: PLC0415
        from ..models.compare_block import CompareBlock # noqa: PLC0415
        before = self.before

        from_version = self.from_version

        key = self.key

        number = self.number

        state = self.state.value

        changed: dict[str, Any] | Unset = UNSET
        if not isinstance(self.changed, Unset):
            changed = self.changed.to_dict()

        source: dict[str, Any] | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.to_dict()

        version: dict[str, Any] | Unset = UNSET
        if not isinstance(self.version, Unset):
            version = self.version.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "before": before,
            "fromVersion": from_version,
            "key": key,
            "number": number,
            "state": state,
        })
        if changed is not UNSET:
            field_dict["changed"] = changed
        if source is not UNSET:
            field_dict["source"] = source
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.block_change import BlockChange # noqa: PLC0415
        from ..models.compare_block import CompareBlock # noqa: PLC0415
        d = dict(src_dict)
        before = d.pop("before")

        from_version = d.pop("fromVersion")

        key = d.pop("key")

        number = d.pop("number")

        state = CompareRowState(d.pop("state"))




        _changed = d.pop("changed", UNSET)
        changed: BlockChange | Unset
        if isinstance(_changed,  Unset):
            changed = UNSET
        else:
            changed = BlockChange.from_dict(_changed)




        _source = d.pop("source", UNSET)
        source: CompareBlock | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = CompareBlock.from_dict(_source)




        _version = d.pop("version", UNSET)
        version: CompareBlock | Unset
        if isinstance(_version,  Unset):
            version = UNSET
        else:
            version = CompareBlock.from_dict(_version)




        compare_row = cls(
            before=before,
            from_version=from_version,
            key=key,
            number=number,
            state=state,
            changed=changed,
            source=source,
            version=version,
        )

        return compare_row

