from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_update_input_source import ImportUpdateInputSource
from ..models.import_update_input_step import ImportUpdateInputStep
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.import_options import ImportOptions





T = TypeVar("T", bound="ImportUpdateInput")



@_attrs_define
class ImportUpdateInput:
    """ 
        Attributes:
            options (ImportOptions | Unset):
            source (ImportUpdateInputSource | Unset):
            step (ImportUpdateInputStep | Unset):
     """

    options: ImportOptions | Unset = UNSET
    source: ImportUpdateInputSource | Unset = UNSET
    step: ImportUpdateInputStep | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_options import ImportOptions # noqa: PLC0415
        options: dict[str, Any] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = self.options.to_dict()

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value


        step: str | Unset = UNSET
        if not isinstance(self.step, Unset):
            step = self.step.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if options is not UNSET:
            field_dict["options"] = options
        if source is not UNSET:
            field_dict["source"] = source
        if step is not UNSET:
            field_dict["step"] = step

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_options import ImportOptions # noqa: PLC0415
        d = dict(src_dict)
        _options = d.pop("options", UNSET)
        options: ImportOptions | Unset
        if isinstance(_options,  Unset):
            options = UNSET
        else:
            options = ImportOptions.from_dict(_options)




        _source = d.pop("source", UNSET)
        source: ImportUpdateInputSource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = ImportUpdateInputSource(_source)




        _step = d.pop("step", UNSET)
        step: ImportUpdateInputStep | Unset
        if isinstance(_step,  Unset):
            step = UNSET
        else:
            step = ImportUpdateInputStep(_step)




        import_update_input = cls(
            options=options,
            source=source,
            step=step,
        )

        return import_update_input

